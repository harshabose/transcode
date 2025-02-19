package transcode

import "github.com/asticode/go-astiav"

const (
	DefaultVideoPayloadType  = 96
	DefaultVideoFPS          = int(25)
	DefaultVideoTimeBase     = int(9000)
	DefaultVideoClockRate    = int(90000)
	DefaultVideoHeight       = int(1080)
	DefaultVideoWidth        = int(1920)
	DefaultVideoPixFormat    = astiav.PixelFormatYuv420P
	DefaultVideoEncoderCodec = astiav.CodecIDH264
)

const (
	DefaultAudioPayloadType  = 111
	DefaultAudioSampleRate   = int(48000)
	DefaultAudioFrameSize    = int(960)
	DefaultAudioSampleFormat = astiav.SampleFormatS16
	DefaultAudioEncoderCodec = astiav.CodecIDOpus
)
