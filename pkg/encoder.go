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

type Encoder struct {
	buffer          buffer.BufferWithGenerator[astiav.Packet]
	filter          *Filter
	ctx             context.Context
	codec           *astiav.Codec
	encoderContext  *astiav.CodecContext
	codecFlags      *astiav.Dictionary
	encoderSettings codecSettings
	bandwidthChan   chan int64
	sps             []byte
	pps             []byte
}

func CreateEncoder(ctx context.Context, codecID astiav.CodecID, filter *Filter, options ...EncoderOption) (*Encoder, error) {
	encoder := &Encoder{
		filter:     filter,
		codecFlags: astiav.NewDictionary(),
		ctx:        ctx,
	}

	encoder.codec = astiav.FindEncoder(codecID)
	if encoder.encoderContext = astiav.AllocCodecContext(encoder.codec); encoder.encoderContext == nil {
		return nil, ErrorAllocateCodecContext
	}

	var contextOption EncoderOption
	if filter.sinkContext.MediaType() == astiav.MediaTypeVideo {
		contextOption = withAudioSetEncoderContextParameters(filter)
	}
	if filter.sinkContext.MediaType() == astiav.MediaTypeVideo {
		contextOption = withVideoSetEncoderContextParameters(filter)
	}

	options = append([]EncoderOption{contextOption}, options...)

	for _, option := range options {
		if err := option(encoder); err != nil {
			return nil, err
		}
	}

	if encoder.encoderSettings == nil {
		return nil, ErrorCodecNoSetting
	}

	encoder.encoderContext.SetFlags(astiav.NewCodecContextFlags(astiav.CodecContextFlagGlobalHeader))

	if err := encoder.encoderContext.Open(encoder.codec, encoder.codecFlags); err != nil {
		return nil, err
	}

	if encoder.buffer == nil {
		encoder.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreatePacketPool())
	}

	return encoder, nil
}

func (encoder *Encoder) Start() {
	go encoder.loop()
}

func (encoder *Encoder) GetDuration() time.Duration {
	if encoder.encoderContext.MediaType() == astiav.MediaTypeAudio {
		return time.Second * time.Duration(encoder.encoderContext.FrameSize()) / time.Duration(encoder.encoderContext.SampleRate())
	}
	return time.Second / time.Duration(encoder.encoderContext.Framerate().Float64())
}

func (encoder *Encoder) GetVideoTimeBase() astiav.Rational {
	return encoder.encoderContext.TimeBase()
}

func (encoder *Encoder) loop() {
	var (
		frame  *astiav.Frame
		packet *astiav.Packet
		err    error
	)
	defer encoder.close()

loop1:
	for {
		select {
		case <-encoder.ctx.Done():
			return
		case bitrate := <-encoder.bandwidthChan: // TODO: MIGHT NEED A MUTEX FOR THIS ONE CASE
			encoder.encoderContext.SetBitRate(bitrate)
			fmt.Printf("bitrate set: %d\n", bitrate)
		case frame = <-encoder.filter.WaitForFrame():
			if err = encoder.encoderContext.SendFrame(frame); err != nil {
				encoder.filter.PutBack(frame)
				if !errors.Is(err, astiav.ErrEagain) {
					continue loop1
				}
			}
		loop2:
			for {
				packet = encoder.buffer.Generate()
				if err = encoder.encoderContext.ReceivePacket(packet); err != nil {
					encoder.buffer.PutBack(packet)
					break loop2
				}

				if err = encoder.pushPacket(packet); err != nil {
					encoder.buffer.PutBack(packet)
					continue loop2
				}
			}
			encoder.filter.PutBack(frame)
		}
	}
}

func (encoder *Encoder) WaitForPacket() chan *astiav.Packet {
	return encoder.buffer.GetChannel()
}

func (encoder *Encoder) pushPacket(packet *astiav.Packet) error {
	ctx, cancel := context.WithTimeout(encoder.ctx, time.Second)
	defer cancel()

	return encoder.buffer.Push(ctx, packet)
}

func (encoder *Encoder) GetPacket() (*astiav.Packet, error) {
	ctx, cancel := context.WithTimeout(encoder.ctx, time.Second)
	defer cancel()

	return encoder.buffer.Pop(ctx)
}

func (encoder *Encoder) PutBack(packet *astiav.Packet) {
	encoder.buffer.PutBack(packet)
}

func (encoder *Encoder) SetBitrateChannel(channel chan int64) {
	encoder.bandwidthChan = channel
}

func (encoder *Encoder) close() {
	if encoder.encoderContext != nil {
		encoder.encoderContext.Free()
	}
}
