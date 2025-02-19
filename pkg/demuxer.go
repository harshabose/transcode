package transcode

import (
	"context"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type Demuxer struct {
	formatContext   *astiav.FormatContext
	inputOptions    *astiav.Dictionary
	inputFormat     *astiav.InputFormat
	stream          *astiav.Stream
	codecParameters *astiav.CodecParameters
	buffer          buffer.BufferWithGenerator[astiav.Packet]
	ctx             context.Context
}

func CreateDemuxer(ctx context.Context, containerAddress string, options ...DemuxerOption) (*Demuxer, error) {
	astiav.RegisterAllDevices()
	demuxer := &Demuxer{
		formatContext: astiav.AllocFormatContext(),
		inputOptions:  astiav.NewDictionary(),
		buffer:        buffer.CreateChannelBuffer(ctx, DefaultVideoFPS*3, internal.CreatePacketPool()),
		ctx:           ctx,
	}

	if demuxer.formatContext == nil {
		return nil, ErrorAllocateFormatContext
	}

	for _, option := range options {
		if err := option(demuxer); err != nil {
			return nil, err
		}
	}

	if err := demuxer.formatContext.OpenInput(containerAddress, demuxer.inputFormat, demuxer.inputOptions); err != nil {
		return nil, ErrorOpenInputContainer
	}

	if err := demuxer.formatContext.FindStreamInfo(nil); err != nil {
		return nil, ErrorNoStreamFound
	}

	for _, stream := range demuxer.formatContext.Streams() {
		if stream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			demuxer.stream = stream
			break
		}
	}

	if demuxer.stream == nil {
		return nil, ErrorNoVideoStreamFound
	}
	demuxer.codecParameters = demuxer.stream.CodecParameters()

	return demuxer, nil
}

func (demuxer *Demuxer) Start() {
	go demuxer.loop()
}

func (demuxer *Demuxer) loop() {
	defer demuxer.close()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

loop1:
	for {
		select {
		case <-demuxer.ctx.Done():
			return
		case <-ticker.C:
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

func (demuxer *Demuxer) pushPacket(packet *astiav.Packet) error {
	ctx, cancel := context.WithTimeout(demuxer.ctx, time.Second/time.Duration(DefaultVideoFPS)) // TODO: NEEDS TO BE BASED ON FPS ON INPUT_FORMAT
	defer cancel()

	return demuxer.buffer.Push(ctx, packet)
}

func (demuxer *Demuxer) WaitForPacket() chan *astiav.Packet {
	return demuxer.buffer.GetChannel()
}

func (demuxer *Demuxer) GetPacket() (*astiav.Packet, error) {
	ctx, cancel := context.WithTimeout(demuxer.ctx, time.Second/time.Duration(DefaultVideoFPS))
	defer cancel()

	return demuxer.buffer.Pop(ctx)
}

func (demuxer *Demuxer) GetDecoderContextOptions() func(*Decoder) error {
	return VideoSetDecoderContext(demuxer.codecParameters, demuxer.stream, demuxer.formatContext)
}

func (demuxer *Demuxer) PutBack(packet *astiav.Packet) {
	demuxer.buffer.PutBack(packet)
}

func (demuxer *Demuxer) close() {
	if demuxer.formatContext != nil {
		demuxer.formatContext.CloseInput()
		demuxer.formatContext.Free()
	}
}
