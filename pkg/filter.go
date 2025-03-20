package transcode

import (
	"context"
	"fmt"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type Filter struct {
	content          string
	decoder          *Decoder
	buffer           buffer.BufferWithGenerator[astiav.Frame]
	graph            *astiav.FilterGraph
	input            *astiav.FilterInOut
	output           *astiav.FilterInOut
	srcContext       *astiav.BuffersrcFilterContext
	sinkContext      *astiav.BuffersinkFilterContext
	srcContextParams *astiav.BuffersrcFilterContextParameters // NOTE: THIS BECOMES NIL AFTER INITIALISATION
	ctx              context.Context
}

func CreateFilter(ctx context.Context, decoder *Decoder, filterConfig *FilterConfig, options ...FilterOption) (*Filter, error) {
	var (
		filter        *Filter
		filterSrc     *astiav.Filter
		filterSink    *astiav.Filter
		contextOption FilterOption
		err           error
	)
	filter = &Filter{
		graph:            astiav.AllocFilterGraph(),
		decoder:          decoder,
		input:            astiav.AllocFilterInOut(),
		output:           astiav.AllocFilterInOut(),
		srcContextParams: astiav.AllocBuffersrcFilterContextParameters(),
		ctx:              ctx,
	}

	// TODO: CHECK IF ALL ATTRIBUTES ARE ALLOCATED PROPERLY

	if filterSrc = astiav.FindFilterByName(filterConfig.Source.String()); filterSrc == nil {
		return nil, ErrorNoFilterName
	}
	if filterSink = astiav.FindFilterByName(filterConfig.Sink.String()); filterSink == nil {
		return nil, ErrorNoFilterName
	}

	if filter.srcContext, err = filter.graph.NewBuffersrcFilterContext(filterSrc, "in"); err != nil {
		return nil, ErrorAllocSrcContext
	}

	if filter.sinkContext, err = filter.graph.NewBuffersinkFilterContext(filterSink, "out"); err != nil {
		return nil, ErrorAllocSinkContext
	}

	if decoder.decoderContext.MediaType() == astiav.MediaTypeVideo {
		fmt.Println("video media type detected")
		contextOption = withVideoSetFilterContextParameters(decoder)
	}
	if decoder.decoderContext.MediaType() == astiav.MediaTypeAudio {
		fmt.Println("audio media type detected")
		contextOption = withAudioSetFilterContextParameters(decoder)
	}

	options = append([]FilterOption{contextOption}, options...)

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

	fmt.Println("check1")

	if err = filter.srcContext.Initialize(astiav.NewDictionary()); err != nil {
		return nil, ErrorSrcContextInitialise
	}

	fmt.Println("check2")

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

func (filter *Filter) Start() {
	go filter.loop()
}

func (filter *Filter) loop() {
	var (
		err       error = nil
		srcFrame  *astiav.Frame
		sinkFrame *astiav.Frame
	)
	defer filter.close()

loop1:
	for {
		select {
		case <-filter.ctx.Done():
			return
		case srcFrame = <-filter.decoder.WaitForFrame():
			if err = filter.srcContext.AddFrame(srcFrame, astiav.NewBuffersrcFlags(astiav.BuffersrcFlagKeepRef)); err != nil {
				filter.buffer.PutBack(srcFrame)
				continue loop1
			}
		loop2:
			for {
				sinkFrame = filter.buffer.Generate()
				if err = filter.sinkContext.GetFrame(sinkFrame, astiav.NewBuffersinkFlags()); err != nil {
					filter.buffer.PutBack(sinkFrame)
					break loop2
				}

				if err = filter.pushFrame(sinkFrame); err != nil {
					filter.buffer.PutBack(sinkFrame)
					continue loop2
				}
			}
			filter.decoder.PutBack(srcFrame)
		}
	}
}

func (filter *Filter) pushFrame(frame *astiav.Frame) error {
	ctx, cancel := context.WithTimeout(filter.ctx, time.Second)
	defer cancel()

	return filter.buffer.Push(ctx, frame)
}

func (filter *Filter) GetFrame() (*astiav.Frame, error) {
	ctx, cancel := context.WithTimeout(filter.ctx, time.Second)
	defer cancel()

	return filter.buffer.Pop(ctx)
}

func (filter *Filter) PutBack(frame *astiav.Frame) {
	filter.buffer.PutBack(frame)
}

func (filter *Filter) WaitForFrame() chan *astiav.Frame {
	return filter.buffer.GetChannel()
}

func (filter *Filter) close() {
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
