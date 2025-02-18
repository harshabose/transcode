package pkg

import "github.com/asticode/go-astiav"

const (
	DefaultVideoPayloadType  = 96
	DefaultVideoFPS          = int(25)
	DefaultVideoClockRate    = int(90000)
	DefaultVideoHeight       = int(1080)
	DefaultVideoWidth        = int(1920)
	DefaultVideoPixFormat    = astiav.PixelFormatYuv420P
	DefaultVideoEncoderCodec = astiav.CodecIDH264
)
