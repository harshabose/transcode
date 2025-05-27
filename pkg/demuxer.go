package transcode

import (
	"context"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
	"github.com/harshabose/tools/buffer/pkg"
)

type GeneralDemuxer struct {
	formatContext   *astiav.FormatContext
	inputOptions    *astiav.Dictionary
	inputFormat     *astiav.InputFormat
	stream          *astiav.Stream
	codecParameters *astiav.CodecParameters
	buffer          buffer.BufferWithGenerator[astiav.Packet]
	ctx             context.Context
	cancel          context.CancelFunc
}

func CreateGeneralDemuxer(ctx context.Context, containerAddress string, options ...DemuxerOption) (*GeneralDemuxer, error) {
	ctx2, cancel := context.WithCancel(ctx)
	astiav.RegisterAllDevices()
	demuxer := &GeneralDemuxer{
		formatContext: astiav.AllocFormatContext(),
		inputOptions:  astiav.NewDictionary(),
		ctx:           ctx2,
		cancel:        cancel,
	}

	if demuxer.formatContext == nil {
		return nil, ErrorAllocateFormatContext
	}

	if demuxer.inputOptions == nil {
		return nil, ErrorGeneralAllocate
	}

	for _, option := range options {
		if err := option(demuxer); err != nil {
			return nil, err
		}
	}

	if err := demuxer.formatContext.OpenInput(containerAddress, demuxer.inputFormat, demuxer.inputOptions); err != nil {
		return nil, err
	}

	if err := demuxer.formatContext.FindStreamInfo(nil); err != nil {
		return nil, ErrorNoStreamFound
	}

	for _, stream := range demuxer.formatContext.Streams() {
		demuxer.stream = stream
		break
	}

	if demuxer.stream == nil {
		return nil, ErrorNoVideoStreamFound
	}
	demuxer.codecParameters = demuxer.stream.CodecParameters()

	if demuxer.buffer == nil {
		demuxer.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreatePacketPool())
	}

	return demuxer, nil
}

func (demuxer *GeneralDemuxer) Ctx() context.Context {
	return demuxer.ctx
}

func (demuxer *GeneralDemuxer) Start() {
	go demuxer.loop()
}

func (demuxer *GeneralDemuxer) Stop() {
	demuxer.cancel()
}

func (demuxer *GeneralDemuxer) loop() {
	defer demuxer.close()

loop1:
	for {
		select {
		case <-demuxer.ctx.Done():
			return
		default:
		loop2:
			for {
				packet := demuxer.buffer.Generate()

				if err := demuxer.formatContext.ReadFrame(packet); err != nil {
					demuxer.buffer.PutBack(packet)
					continue loop1
				}

				if packet.StreamIndex() != demuxer.stream.Index() {
					demuxer.buffer.PutBack(packet)
					continue loop2
				}

				if err := demuxer.pushPacket(packet); err != nil {
					demuxer.buffer.PutBack(packet)
					continue loop1
				}
				break loop2
			}
		}
	}
}

func (demuxer *GeneralDemuxer) pushPacket(packet *astiav.Packet) error {
	ctx, cancel := context.WithTimeout(demuxer.ctx, time.Second) // TODO: NEEDS TO BE BASED ON FPS ON INPUT_FORMAT
	defer cancel()

	return demuxer.buffer.Push(ctx, packet)
}

func (demuxer *GeneralDemuxer) WaitForPacket() chan *astiav.Packet {
	return demuxer.buffer.GetChannel()
}

func (demuxer *GeneralDemuxer) GetPacket() (*astiav.Packet, error) {
	ctx, cancel := context.WithTimeout(demuxer.ctx, time.Second)
	defer cancel()

	return demuxer.buffer.Pop(ctx)
}

func (demuxer *GeneralDemuxer) PutBack(packet *astiav.Packet) {
	demuxer.buffer.PutBack(packet)
}

func (demuxer *GeneralDemuxer) close() {
	if demuxer.formatContext != nil {
		demuxer.formatContext.CloseInput()
		demuxer.formatContext.Free()
	}
}

func (demuxer *GeneralDemuxer) SetInputOption(key, value string, flags astiav.DictionaryFlags) error {
	return demuxer.inputOptions.Set(key, value, flags)
}

func (demuxer *GeneralDemuxer) SetInputFormat(format *astiav.InputFormat) {
	demuxer.inputFormat = format
}

func (demuxer *GeneralDemuxer) SetBuffer(buffer buffer.BufferWithGenerator[astiav.Packet]) {
	demuxer.buffer = buffer
}

func (demuxer *GeneralDemuxer) GetCodecParameters() *astiav.CodecParameters {
	return demuxer.codecParameters
}

func (demuxer *GeneralDemuxer) MediaType() astiav.MediaType {
	return demuxer.codecParameters.MediaType()
}

func (demuxer *GeneralDemuxer) CodecID() astiav.CodecID {
	return demuxer.codecParameters.CodecID()
}

func (demuxer *GeneralDemuxer) FrameRate() astiav.Rational {
	return demuxer.formatContext.GuessFrameRate(demuxer.stream, nil)
}

func (demuxer *GeneralDemuxer) TimeBase() astiav.Rational {
	return demuxer.stream.TimeBase()
}
