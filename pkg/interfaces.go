package transcode

import (
	"context"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"
)

type CanSetDemuxerInputOption interface {
	SetInputOption(key, value string, flags astiav.DictionaryFlags) error
}

type CanSetDemuxerInputFormat interface {
	SetInputFormat(*astiav.InputFormat)
}

type CanSetBuffer[T any] interface {
	SetBuffer(buffer buffer.BufferWithGenerator[T])
}

type CanDescribeFrameRate interface {
	FrameRate() astiav.Rational
}

type CanDescribeTimeBase interface {
	TimeBase() astiav.Rational
}

type CanSetFrameRate interface {
	SetFrameRate(CanDescribeFrameRate)
}

type CanSetTimeBase interface {
	SetTimeBase(CanDescribeTimeBase)
}

type CanDescribeMediaPacket interface {
	MediaType() astiav.MediaType
	CodecID() astiav.CodecID
	GetCodecParameters() *astiav.CodecParameters
	CanDescribeFrameRate
	CanDescribeTimeBase
}

type CanProduceMediaPacket interface {
	GetPacket(ctx context.Context) (*astiav.Packet, error)
	PutBack(*astiav.Packet)
}

type CanProduceMediaFrame interface {
	GetFrame(ctx context.Context) (*astiav.Frame, error)
	PutBack(*astiav.Frame)
}

type CanDescribeMediaVideoFrame interface {
	CanDescribeFrameRate
	CanDescribeTimeBase
	Height() int
	Width() int
	PixelFormat() astiav.PixelFormat
	SampleAspectRatio() astiav.Rational
	ColorSpace() astiav.ColorSpace
	ColorRange() astiav.ColorRange
}

type CanSetMediaVideoFrame interface {
	CanSetFrameRate
	CanSetTimeBase
	SetHeight(CanDescribeMediaVideoFrame)
	SetWidth(CanDescribeMediaVideoFrame)
	SetPixelFormat(CanDescribeMediaVideoFrame)
	SetSampleAspectRatio(CanDescribeMediaVideoFrame)
	SetColorSpace(CanDescribeMediaVideoFrame)
	SetColorRange(CanDescribeMediaVideoFrame)
}

type CanDescribeMediaFrame interface {
	MediaType() astiav.MediaType
	CanDescribeMediaVideoFrame
	CanDescribeMediaAudioFrame
}

type CanSetMediaAudioFrame interface {
	CanSetTimeBase
	SetSampleRate(CanDescribeMediaAudioFrame)
	SetSampleFormat(CanDescribeMediaAudioFrame)
	SetChannelLayout(CanDescribeMediaAudioFrame)
}

type CanDescribeMediaAudioFrame interface {
	CanDescribeTimeBase
	SampleRate() int
	SampleFormat() astiav.SampleFormat
	ChannelLayout() astiav.ChannelLayout
}

type CanSetMediaPacket interface {
	FillContextContent(CanDescribeMediaPacket) error
	SetCodec(CanDescribeMediaPacket) error
	CanSetFrameRate
	CanSetTimeBase
}

type Demuxer interface {
	Ctx() context.Context
	Start()
	Stop()
	CanProduceMediaPacket
}

type Decoder interface {
	Ctx() context.Context
	Start()
	Stop()
	CanProduceMediaFrame
}

type CanAddToFilterContent interface {
	AddToFilterContent(string)
}

type Filter interface {
	Ctx() context.Context
	Start()
	Stop()
	CanProduceMediaFrame
}

type CanPauseUnPauseEncoder interface {
	PauseEncoding() error
	UnPauseEncoding() error
}

type CanGetParameterSets interface {
	GetParameterSets() (sps, pps []byte, err error)
}

type Encoder interface {
	Ctx() context.Context
	Start()
	Stop()
	CanProduceMediaPacket
}

type CanSetEncoderCodecSettings interface {
	SetEncoderCodecSettings(codecSettings) error
}

type CanUpdateBitrate interface {
	UpdateBitrate(int64) error
}

type CanGetCurrentBitrate interface {
	GetCurrentBitrate() (int64, error)
}

type UpdateBitrateCallBack func(bps int64) error

type CanGetUpdateBitrateCallBack interface {
	OnUpdateBitrate() UpdateBitrateCallBack
}
