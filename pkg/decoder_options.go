package pkg

import "github.com/asticode/go-astiav"

type DecoderOption = func(*Decoder) error

func VideoSetDecoderContext(codecParameters *astiav.CodecParameters, videoStream *astiav.Stream, formatContext *astiav.FormatContext) func(*Decoder) error {
	return func(decoder *Decoder) error {
		var (
			err error
		)

		if decoder.codec = astiav.FindDecoder(codecParameters.CodecID()); decoder.codec == nil {
			return ErrorNoCodecFound
		}

		if decoder.decoderContext = astiav.AllocCodecContext(decoder.codec); decoder.decoderContext == nil {
			return ErrorAllocateCodecContext
		}

		if err = videoStream.CodecParameters().ToCodecContext(decoder.decoderContext); err != nil {
			return ErrorFillCodecContext
		}

		decoder.decoderContext.SetFramerate(formatContext.GuessFrameRate(videoStream, nil))
		decoder.decoderContext.SetTimeBase(videoStream.TimeBase())
		return nil
	}
}
