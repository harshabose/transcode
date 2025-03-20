package transcode

import (
	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type DemuxerOption = func(*Demuxer) error

func WithRTSPInputOption(demuxer *Demuxer) error {
	var err error = nil

	if err = demuxer.inputOptions.Set("rtsp_transport", "tcp", 0); err != nil {
		return err
	}
	if err = demuxer.inputOptions.Set("stimeout", "5000000", 0); err != nil {
		return err
	}
	if err = demuxer.inputOptions.Set("fflags", "nobuffer", 0); err != nil {
		return err
	}
	if err = demuxer.inputOptions.Set("flags", "low_delay", 0); err != nil {
		return err
	}
	if err = demuxer.inputOptions.Set("reorder_queue_size", "0", 0); err != nil {
		return err
	}

	return nil
}

func WithAlsaInputFormatOption(demuxer *Demuxer) error {
	demuxer.inputFormat = astiav.FindInputFormat("alsa")
	return nil
}

func WithAvFoundationInputFormatOption(demuxer *Demuxer) error {
	demuxer.inputFormat = astiav.FindInputFormat("avfoundation")
	return nil
}

func WithDemuxerBufferSize(size int) DemuxerOption {
	return func(demuxer *Demuxer) error {
		demuxer.buffer = buffer.CreateChannelBuffer(demuxer.ctx, size, internal.CreatePacketPool())
		return nil
	}
}
