package transcode

import (
	"context"
	"fmt"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type GeneralFilter struct {
	content          string
	decoder          CanProduceMediaFrame
	buffer           buffer.BufferWithGenerator[astiav.Frame]
	graph            *astiav.FilterGraph
	input            *astiav.FilterInOut
	output           *astiav.FilterInOut
	srcContext       *astiav.BuffersrcFilterContext
	sinkContext      *astiav.BuffersinkFilterContext
	srcContextParams *astiav.BuffersrcFilterContextParameters // NOTE: THIS BECOMES NIL AFTER INITIALISATION
	ctx              context.Context
	cancel           context.CancelFunc
}

func CreateGeneralFilter(ctx context.Context, canProduceMediaFrame CanProduceMediaFrame, filterConfig FilterConfig, options ...FilterOption) (*GeneralFilter, error) {
	ctx2, cancel := context.WithCancel(ctx)
	filter := &GeneralFilter{
		graph:            astiav.AllocFilterGraph(),
		decoder:          canProduceMediaFrame,
		input:            astiav.AllocFilterInOut(),
		output:           astiav.AllocFilterInOut(),
		srcContextParams: astiav.AllocBuffersrcFilterContextParameters(),
		ctx:              ctx2,
		cancel:           cancel,
	}

	// TODO: CHECK IF ALL ATTRIBUTES ARE ALLOCATED PROPERLY

	filterSrc := astiav.FindFilterByName(filterConfig.Source.String())
	if filterSrc == nil {
		return nil, ErrorNoFilterName
	}

	filterSink := astiav.FindFilterByName(filterConfig.Sink.String())
	if filterSink == nil {
		return nil, ErrorNoFilterName
	}

	srcContext, err := filter.graph.NewBuffersrcFilterContext(filterSrc, "in")
	if err != nil {
		return nil, ErrorAllocSrcContext
	}
	filter.srcContext = srcContext

	sinkContext, err := filter.graph.NewBuffersinkFilterContext(filterSink, "out")
	if err != nil {
		return nil, ErrorAllocSinkContext
	}
	filter.sinkContext = sinkContext

	canDescribeMediaFrame, ok := canProduceMediaFrame.(CanDescribeMediaFrame)
	if !ok {
		return nil, ErrorInterfaceMismatch
	}
	if canDescribeMediaFrame.MediaType() == astiav.MediaTypeVideo {
		options = append([]FilterOption{withVideoSetFilterContextParameters(canDescribeMediaFrame)}, options...)
	}
	if canDescribeMediaFrame.MediaType() == astiav.MediaTypeAudio {
		options = append([]FilterOption{withAudioSetFilterContextParameters(canDescribeMediaFrame)}, options...)
	}

	for _, option := range options {
		if err = option(filter); err != nil {
			// TODO: SET CONTENT HERE
			return nil, err
		}
	}

	if filter.buffer == nil {
		filter.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreateFramePool())
	}

	if err = filter.srcContext.SetParameters(filter.srcContextParams); err != nil {
		return nil, ErrorSrcContextSetParameter
	}

	if err = filter.srcContext.Initialize(astiav.NewDictionary()); err != nil {
		return nil, ErrorSrcContextInitialise
	}

	filter.output.SetName("in")
	filter.output.SetFilterContext(filter.srcContext.FilterContext())
	filter.output.SetPadIdx(0)
	filter.output.SetNext(nil)

	filter.input.SetName("out")
	filter.input.SetFilterContext(filter.sinkContext.FilterContext())
	filter.input.SetPadIdx(0)
	filter.input.SetNext(nil)

	if filter.content == "" {
		fmt.Println(WarnNoFilterContent)
	}

	if err = filter.graph.Parse(filter.content, filter.input, filter.output); err != nil {
		return nil, ErrorGraphParse
	}

	if err = filter.graph.Configure(); err != nil {
		return nil, ErrorGraphConfigure
	}

	if filter.srcContextParams != nil {
		filter.srcContextParams.Free()
	}

	return filter, nil
}

func (filter *GeneralFilter) Ctx() context.Context {
	return filter.ctx
}

func (filter *GeneralFilter) Start() {
	go filter.loop()
}

func (filter *GeneralFilter) Stop() {
	filter.cancel()
}

func (filter *GeneralFilter) loop() {
	defer filter.close()

loop1:
	for {
		select {
		case <-filter.ctx.Done():
			return
		default:
			srcFrame, err := filter.getFrame()
			if err != nil {
				// fmt.Println("unable to get frame from decoder; err:", err.Error())
				continue
			}
			if err := filter.srcContext.AddFrame(srcFrame, astiav.NewBuffersrcFlags(astiav.BuffersrcFlagKeepRef)); err != nil {
				filter.buffer.PutBack(srcFrame)
				continue loop1
			}
		loop2:
			for {
				sinkFrame := filter.buffer.Generate()
				if err = filter.sinkContext.GetFrame(sinkFrame, astiav.NewBuffersinkFlags()); err != nil {
					filter.buffer.PutBack(sinkFrame)
					break loop2
				}

				if err := filter.pushFrame(sinkFrame); err != nil {
					filter.buffer.PutBack(sinkFrame)
					continue loop2
				}
			}
			filter.decoder.PutBack(srcFrame)
		}
	}
}

func (filter *GeneralFilter) pushFrame(frame *astiav.Frame) error {
	ctx, cancel := context.WithTimeout(filter.ctx, 50*time.Millisecond)
	defer cancel()

	return filter.buffer.Push(ctx, frame)
}

func (filter *GeneralFilter) getFrame() (*astiav.Frame, error) {
	ctx, cancel := context.WithTimeout(filter.ctx, 50*time.Millisecond)
	defer cancel()

	return filter.decoder.GetFrame(ctx)
}

func (filter *GeneralFilter) PutBack(frame *astiav.Frame) {
	filter.buffer.PutBack(frame)
}

func (filter *GeneralFilter) GetFrame(ctx context.Context) (*astiav.Frame, error) {
	return filter.buffer.Pop(ctx)
}

func (filter *GeneralFilter) close() {
	if filter.graph != nil {
		filter.graph.Free()
	}
	if filter.input != nil {
		filter.input.Free()
	}
	if filter.output != nil {
		filter.output.Free()
	}
}

func (filter *GeneralFilter) SetBuffer(buffer buffer.BufferWithGenerator[astiav.Frame]) {
	filter.buffer = buffer
}

func (filter *GeneralFilter) AddToFilterContent(content string) {
	filter.content += content
}

func (filter *GeneralFilter) SetFrameRate(describe CanDescribeFrameRate) {
	filter.srcContextParams.SetFramerate(describe.FrameRate())
}

func (filter *GeneralFilter) SetTimeBase(describe CanDescribeTimeBase) {
	filter.srcContextParams.SetTimeBase(describe.TimeBase())
}

func (filter *GeneralFilter) SetHeight(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetHeight(describe.Height())
}

func (filter *GeneralFilter) SetWidth(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetWidth(describe.Width())
}

func (filter *GeneralFilter) SetPixelFormat(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetPixelFormat(describe.PixelFormat())
}

func (filter *GeneralFilter) SetSampleAspectRatio(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetSampleAspectRatio(describe.SampleAspectRatio())
}

func (filter *GeneralFilter) SetColorSpace(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetColorSpace(describe.ColorSpace())
}

func (filter *GeneralFilter) SetColorRange(describe CanDescribeMediaVideoFrame) {
	filter.srcContextParams.SetColorRange(describe.ColorRange())
}

func (filter *GeneralFilter) SetSampleRate(describe CanDescribeMediaAudioFrame) {
	filter.srcContextParams.SetSampleRate(describe.SampleRate())
}

func (filter *GeneralFilter) SetSampleFormat(describe CanDescribeMediaAudioFrame) {
	filter.srcContextParams.SetSampleFormat(describe.SampleFormat())
}

func (filter *GeneralFilter) SetChannelLayout(describe CanDescribeMediaAudioFrame) {
	filter.srcContextParams.SetChannelLayout(describe.ChannelLayout())
}

func (filter *GeneralFilter) MediaType() astiav.MediaType {
	return filter.sinkContext.MediaType()
}

func (filter *GeneralFilter) FrameRate() astiav.Rational {
	return filter.sinkContext.FrameRate()
}

func (filter *GeneralFilter) TimeBase() astiav.Rational {
	return filter.sinkContext.TimeBase()
}

func (filter *GeneralFilter) Height() int {
	return filter.sinkContext.Height()
}

func (filter *GeneralFilter) Width() int {
	return filter.sinkContext.Width()
}

func (filter *GeneralFilter) PixelFormat() astiav.PixelFormat {
	return filter.sinkContext.PixelFormat()
}

func (filter *GeneralFilter) SampleAspectRatio() astiav.Rational {
	return filter.sinkContext.SampleAspectRatio()
}

func (filter *GeneralFilter) ColorSpace() astiav.ColorSpace {
	return filter.sinkContext.ColorSpace()
}

func (filter *GeneralFilter) ColorRange() astiav.ColorRange {
	return filter.sinkContext.ColorRange()
}

func (filter *GeneralFilter) SampleRate() int {
	return filter.sinkContext.SampleRate()
}

func (filter *GeneralFilter) SampleFormat() astiav.SampleFormat {
	return filter.sinkContext.SampleFormat()
}

func (filter *GeneralFilter) ChannelLayout() astiav.ChannelLayout {
	return filter.sinkContext.ChannelLayout()
}
