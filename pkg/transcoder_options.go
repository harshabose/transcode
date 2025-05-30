package transcode

import (
	"context"

	"github.com/asticode/go-astiav"
)

type TranscoderOption = func(*Transcoder) error

func WithGeneralDemuxer(ctx context.Context, containerAddress string, options ...DemuxerOption) TranscoderOption {
	return func(transcoder *Transcoder) error {
		demuxer, err := CreateGeneralDemuxer(ctx, containerAddress, options...)
		if err != nil {
			return err
		}

		transcoder.demuxer = demuxer
		return nil
	}
}

func WithGeneralDecoder(ctx context.Context, options ...DecoderOption) TranscoderOption {
	return func(transcoder *Transcoder) error {
		decoder, err := CreateGeneralDecoder(ctx, transcoder.demuxer, options...)
		if err != nil {
			return err
		}

		transcoder.decoder = decoder
		return nil
	}
}

func WithGeneralFilter(ctx context.Context, filterConfig FilterConfig, options ...FilterOption) TranscoderOption {
	return func(transcoder *Transcoder) error {
		filter, err := CreateGeneralFilter(ctx, transcoder.decoder, filterConfig, options...)
		if err != nil {
			return err
		}

		transcoder.filter = filter
		return nil
	}
}

func WithGeneralEncoder(ctx context.Context, codecID astiav.CodecID, options ...EncoderOption) TranscoderOption {
	return func(transcoder *Transcoder) error {
		encoder, err := CreateGeneralEncoder(ctx, codecID, transcoder.filter, options...)
		if err != nil {
			return err
		}

		transcoder.encoder = encoder
		return nil
	}
}

func WithBitrateControlEncoder(ctx context.Context, codecID astiav.CodecID, bitrateControlConfig UpdateConfig, settings codecSettings, bufferSize int) TranscoderOption {
	return func(transcoder *Transcoder) error {
		builder := NewEncoderBuilder(codecID, settings, bufferSize, transcoder.filter)
		updateEncoder, err := NewUpdateEncoder(ctx, bitrateControlConfig, builder)
		if err != nil {
			return err
		}

		transcoder.encoder = updateEncoder
		return nil
	}
}

func WithMultiEncoderBitrateControl(ctx context.Context, codecID astiav.CodecID, config MultiConfig, settings codecSettings, bufferSize int) TranscoderOption {
	return func(transcoder *Transcoder) error {
		builder := NewEncoderBuilder(codecID, settings, bufferSize, transcoder.filter)
		multiEncoder, err := NewMultiUpdateEncoder(ctx, config, builder)
		if err != nil {
			return err
		}

		transcoder.encoder = multiEncoder
		return nil
	}
}
