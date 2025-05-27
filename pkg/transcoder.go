package transcode

import (
	"github.com/asticode/go-astiav"
)

type Transcoder struct {
	demuxer Demuxer
	decoder Decoder
	filter  Filter
	encoder Encoder
}

func CreateTranscoder(options ...TranscoderOption) (*Transcoder, error) {
	t := &Transcoder{}
	for _, option := range options {
		if err := option(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

func NewTranscoder(demuxer Demuxer, decoder Decoder, filter Filter, encoder Encoder) *Transcoder {
	return &Transcoder{
		demuxer: demuxer,
		decoder: decoder,
		filter:  filter,
		encoder: encoder,
	}
}

func (t *Transcoder) Start() {
	t.demuxer.Start()
	t.decoder.Start()
	t.filter.Start()
	t.encoder.Start()
}

func (t *Transcoder) Stop() {
	t.encoder.Stop()
	t.filter.Stop()
	t.decoder.Stop()
	t.demuxer.Stop()
}

func (t *Transcoder) WaitForPacket() chan *astiav.Packet {
	return t.encoder.WaitForPacket()
}

func (t *Transcoder) PutBack(packet *astiav.Packet) {
	t.encoder.PutBack(packet)
}

func (t *Transcoder) PauseEncoding() error {
	p, ok := t.encoder.(CanPauseUnPauseEncoder)
	if !ok {
		return ErrorInterfaceMismatch
	}

	return p.PauseEncoding()
}

func (t *Transcoder) UnPauseEncoding() error {
	p, ok := t.encoder.(CanPauseUnPauseEncoder)
	if !ok {
		return ErrorInterfaceMismatch
	}

	return p.UnPauseEncoding()
}

func (t *Transcoder) GetParameterSets() (sps, pps []byte, err error) {
	p, ok := t.encoder.(CanGetParameterSets)
	if !ok {
		return nil, nil, ErrorInterfaceMismatch
	}

	return p.GetParameterSets()
}

func (t *Transcoder) UpdateBitrate(bps int64) error {
	u, ok := t.encoder.(CanUpdateBitrate)
	if !ok {
		return ErrorInterfaceMismatch
	}

	return u.UpdateBitrate(bps)
}

func (t *Transcoder) OnUpdateBitrate() UpdateBitrateCallBack {
	return t.UpdateBitrate
}
