package pkg

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
