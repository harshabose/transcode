package transcode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/asticode/go-astiav"
	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type Decoder struct {
	demuxer        *Demuxer
	decoderContext *astiav.CodecContext
	codec          *astiav.Codec
	buffer         buffer.BufferWithGenerator[astiav.Frame]
	ctx            context.Context
}

func CreateDecoder(ctx context.Context, demuxer *Demuxer, options ...DecoderOption) (*Decoder, error) {
	var (
		err           error
		contextOption DecoderOption
		decoder       *Decoder
	)

	decoder = &Decoder{
		demuxer: demuxer,
		ctx:     ctx,
	}

	if demuxer.stream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
		contextOption = withVideoSetDecoderContext(demuxer)
	}
	if demuxer.stream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
		contextOption = withVideoSetDecoderContext(demuxer)
	}

	options = append([]DecoderOption{contextOption}, options...)

	for _, option := range options {
		if err = option(decoder); err != nil {
			return nil, err
		}
	}

	if decoder.buffer == nil {
		decoder.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreateFramePool())
	}

	if err = decoder.decoderContext.Open(decoder.codec, nil); err != nil {
		return nil, err
	}

	return decoder, nil
}

func (decoder *Decoder) Start() {
	go decoder.loop()
}

func (decoder *Decoder) loop() {
	var (
		packet *astiav.Packet
		frame  *astiav.Frame
		err    error
	)

	defer decoder.close()

loop1:
	for {
		select {
		case <-decoder.ctx.Done():
			return
		case packet = <-decoder.demuxer.WaitForPacket():
			if err := decoder.decoderContext.SendPacket(packet); err != nil {
				decoder.demuxer.PutBack(packet)
				if !errors.Is(err, astiav.ErrEagain) {
					continue loop1
				}
			}
		loop2:
			for {
				frame = decoder.buffer.Generate()
				if err := decoder.decoderContext.ReceiveFrame(frame); err != nil {
					decoder.buffer.PutBack(frame)
					break loop2
				}

				frame.SetPictureType(astiav.PictureTypeNone)

				if err = decoder.pushFrame(frame); err != nil {
					fmt.Println("warning: frame dropped!")
					decoder.buffer.PutBack(frame)
					continue loop2
				}
			}
			decoder.demuxer.PutBack(packet)
		}
	}
}

func (decoder *Decoder) pushFrame(frame *astiav.Frame) error {
	ctx, cancel := context.WithTimeout(decoder.ctx, time.Second)
	defer cancel()

	return decoder.buffer.Push(ctx, frame)
}

func (decoder *Decoder) GetFrame() (*astiav.Frame, error) {
	ctx, cancel := context.WithTimeout(decoder.ctx, time.Second)
	defer cancel()

	return decoder.buffer.Pop(ctx)
}

func (decoder *Decoder) WaitForFrame() chan *astiav.Frame {
	return decoder.buffer.GetChannel()
}

func (decoder *Decoder) PutBack(frame *astiav.Frame) {
	decoder.buffer.PutBack(frame)
}

func (decoder *Decoder) close() {
	if decoder.decoderContext != nil {
		decoder.decoderContext.Free()
	}
}
