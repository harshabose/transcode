package transcode

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type GeneralEncoder struct {
	buffer          buffer.BufferWithGenerator[astiav.Packet]
	producer        CanProduceMediaFrame
	codec           *astiav.Codec
	encoderContext  *astiav.CodecContext
	codecFlags      *astiav.Dictionary
	encoderSettings codecSettings
	sps             []byte
	pps             []byte
	ctx             context.Context
	cancel          context.CancelFunc
}

func CreateGeneralEncoder(ctx context.Context, codecID astiav.CodecID, canProduceMediaFrame CanProduceMediaFrame, options ...EncoderOption) (*GeneralEncoder, error) {
	ctx2, cancel := context.WithCancel(ctx)
	encoder := &GeneralEncoder{
		producer:   canProduceMediaFrame,
		codecFlags: astiav.NewDictionary(),
		ctx:        ctx2,
		cancel:     cancel,
	}

	encoder.codec = astiav.FindEncoder(codecID)
	if encoder.encoderContext = astiav.AllocCodecContext(encoder.codec); encoder.encoderContext == nil {
		return nil, ErrorAllocateCodecContext
	}

	canDescribeMediaFrame, ok := canProduceMediaFrame.(CanDescribeMediaFrame)
	if !ok {
		return nil, ErrorInterfaceMismatch
	}
	if canDescribeMediaFrame.MediaType() == astiav.MediaTypeAudio {
		withAudioSetEncoderContextParameters(canDescribeMediaFrame, encoder.encoderContext)
	}
	if canDescribeMediaFrame.MediaType() == astiav.MediaTypeVideo {
		withVideoSetEncoderContextParameter(canDescribeMediaFrame, encoder.encoderContext)
	}

	for _, option := range options {
		if err := option(encoder); err != nil {
			return nil, err
		}
	}

	if encoder.encoderSettings == nil {
		fmt.Println("warn: no encoder settings are provided")
	}

	encoder.encoderContext.SetFlags(astiav.NewCodecContextFlags(astiav.CodecContextFlagGlobalHeader))

	if err := encoder.encoderContext.Open(encoder.codec, encoder.codecFlags); err != nil {
		return nil, err
	}

	if encoder.buffer == nil {
		encoder.buffer = buffer.CreateChannelBuffer(ctx2, 256, internal.CreatePacketPool())
	}

	encoder.findParameterSets(encoder.encoderContext.ExtraData())

	return encoder, nil
}

func (encoder *GeneralEncoder) Ctx() context.Context {
	return encoder.ctx
}

func (encoder *GeneralEncoder) Start() {
	go encoder.loop()
}

func (encoder *GeneralEncoder) GetParameterSets() ([]byte, []byte, error) {
	encoder.findParameterSets(encoder.encoderContext.ExtraData())
	return encoder.sps, encoder.pps, nil
}

func (encoder *GeneralEncoder) TimeBase() astiav.Rational {
	return encoder.encoderContext.TimeBase()
}

func (encoder *GeneralEncoder) loop() {
	defer encoder.close()

loop1:
	for {
		select {
		case <-encoder.ctx.Done():
			return
		default:
			frame, err := encoder.getFrame()
			if err != nil {
				// fmt.Println("unable to get packet from encoder; err:", err.Error())
				continue
			}
			if err := encoder.encoderContext.SendFrame(frame); err != nil {
				encoder.producer.PutBack(frame)
				if !errors.Is(err, astiav.ErrEagain) {
					continue loop1
				}
			}
		loop2:
			for {
				packet := encoder.buffer.Generate()
				if err = encoder.encoderContext.ReceivePacket(packet); err != nil {
					encoder.buffer.PutBack(packet)
					break loop2
				}

				if err := encoder.pushPacket(packet); err != nil {
					encoder.buffer.PutBack(packet)
					continue loop2
				}
			}
			encoder.producer.PutBack(frame)
		}
	}
}

func (encoder *GeneralEncoder) getFrame() (*astiav.Frame, error) {
	ctx, cancel := context.WithTimeout(encoder.ctx, 50*time.Millisecond)
	defer cancel()

	return encoder.producer.GetFrame(ctx)
}

func (encoder *GeneralEncoder) GetPacket(ctx context.Context) (*astiav.Packet, error) {
	return encoder.buffer.Pop(ctx)
}

func (encoder *GeneralEncoder) pushPacket(packet *astiav.Packet) error {
	ctx, cancel := context.WithTimeout(encoder.ctx, 50*time.Millisecond)
	defer cancel()

	return encoder.buffer.Push(ctx, packet)
}

func (encoder *GeneralEncoder) PutBack(packet *astiav.Packet) {
	encoder.buffer.PutBack(packet)
}

func (encoder *GeneralEncoder) Stop() {
	encoder.cancel()
}

func (encoder *GeneralEncoder) close() {
	if encoder.encoderContext != nil {
		encoder.encoderContext.Free()
	}

	if encoder.codecFlags != nil {
		encoder.codecFlags.Free()
	}
}

func (encoder *GeneralEncoder) findParameterSets(extraData []byte) {
	if len(extraData) > 0 {
		// Find the first start code (0x00000001)
		for i := 0; i < len(extraData)-4; i++ {
			if extraData[i] == 0 && extraData[i+1] == 0 && extraData[i+2] == 0 && extraData[i+3] == 1 {
				// Skip start code to get the NAL type
				nalType := extraData[i+4] & 0x1F

				// Find the next start code or end
				nextStart := len(extraData)
				for j := i + 4; j < len(extraData)-4; j++ {
					if extraData[j] == 0 && extraData[j+1] == 0 && extraData[j+2] == 0 && extraData[j+3] == 1 {
						nextStart = j
						break
					}
				}

				if nalType == 7 { // SPS
					encoder.sps = make([]byte, nextStart-i)
					copy(encoder.sps, extraData[i:nextStart])
				} else if nalType == 8 { // PPS
					encoder.pps = make([]byte, len(extraData)-i)
					copy(encoder.pps, extraData[i:])
				}

				i = nextStart - 1
			}
		}
		fmt.Println("SPS for current encoder: ", encoder.sps)
		fmt.Println("\tSPS for current encoder in Base64:", base64.StdEncoding.EncodeToString(encoder.sps))
		fmt.Println("PPS for current encoder: ", encoder.pps)
		fmt.Println("\tPPS for current encoder in Base64:", base64.StdEncoding.EncodeToString(encoder.pps))
	}
}

func (encoder *GeneralEncoder) SetBuffer(buffer buffer.BufferWithGenerator[astiav.Packet]) {
	encoder.buffer = buffer
}

func (encoder *GeneralEncoder) SetEncoderCodecSettings(settings codecSettings) error {
	encoder.encoderSettings = settings
	return encoder.encoderSettings.ForEach(func(key string, value string) error {
		if value == "" {
			return nil
		}
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func (encoder *GeneralEncoder) GetCurrentBitrate() (int64, error) {
	g, ok := encoder.encoderSettings.(CanGetCurrentBitrate)
	if !ok {
		return 0, ErrorInterfaceMismatch
	}

	return g.GetCurrentBitrate()
}
