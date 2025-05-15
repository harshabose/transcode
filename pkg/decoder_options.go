package transcode

import (
	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type DecoderOption = func(*Decoder) error

func withVideoSetDecoderContext(demuxer *Demuxer) func(*Decoder) error {
	return func(decoder *Decoder) error {
		var (
			err error
		)

		if decoder.codec = astiav.FindDecoder(demuxer.codecParameters.CodecID()); decoder.codec == nil {
			return ErrorNoCodecFound
		}

		if decoder.decoderContext = astiav.AllocCodecContext(decoder.codec); decoder.decoderContext == nil {
			return ErrorAllocateCodecContext
		}

		if err = demuxer.stream.CodecParameters().ToCodecContext(decoder.decoderContext); err != nil {
			return ErrorFillCodecContext
		}

		decoder.decoderContext.SetFramerate(demuxer.formatContext.GuessFrameRate(demuxer.stream, nil))
		decoder.decoderContext.SetTimeBase(demuxer.stream.TimeBase())
		return nil
	}
}

func withAudioSetDecoderContext(demuxer *Demuxer) func(*Decoder) error {
	return func(decoder *Decoder) error {
		var (
			err error
		)

		if decoder.codec = astiav.FindDecoder(demuxer.codecParameters.CodecID()); decoder.codec == nil {
			return ErrorNoCodecFound
		}

		if decoder.decoderContext = astiav.AllocCodecContext(decoder.codec); decoder.decoderContext == nil {
			return ErrorAllocateCodecContext
		}

		if err = demuxer.stream.CodecParameters().ToCodecContext(decoder.decoderContext); err != nil {
			return ErrorFillCodecContext
		}

		decoder.decoderContext.SetTimeBase(demuxer.stream.TimeBase())
		return nil
	}
}

func WithDecoderBufferSize(size int) DecoderOption {
	return func(decoder *Decoder) error {
		decoder.buffer = buffer.CreateChannelBuffer(decoder.ctx, size, internal.CreateFramePool())
		return nil
	}
}
