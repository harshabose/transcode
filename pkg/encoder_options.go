package transcode

import (
	"reflect"

	buffer "github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type (
	EncoderOption = func(*Encoder) error
)

type codecSettings interface {
	ForEach(func(string, string) error) error
}

type X264Settings struct {
	Preset        string `x264:"preset"`
	Tune          string `x264:"tune"`
	Refs          string `x264:"refs"`
	Profile       string `x264:"profile"`
	Level         string `x264:"level"`
	Qmin          string `x264:"qmin"`
	Qmax          string `x264:"qmax"`
	BFrames       string `x264:"bframes"`
	BAdapt        string `x264:"b-adapt"`
	NGOP          string `x264:"keyint"`
	NGOPMin       string `x264:"min-keyint"`
	Scenecut      string `x264:"scenecut"`
	InfraRefresh  string `x264:"intra-refresh"`
	LookAhead     string `x264:"rc-lookahead"`
	SlicedThreads string `x264:"sliced-threads"`
	SyncLookAhead string `x264:"sync-lookahead"`
	ForceIDR      string `x264:"force-idr"`
	AQMode        string `x264:"aq-mode"`
	AQStrength    string `x264:"aq-strength"`
	MBTree        string `x264:"mbtree"`
	Bitrate       string `x264:"bitrate"`
	VBVMaxBitrate string `x264:"vbv-maxrate"`
	VBVBuffer     string `x264:"vbv-bufsize"`
	RateTol       string `x264:"ratetol"`
	Threads       string `x264:"threads"`
	AnnexB        string `x264:"annexb"`
	Aud           string `x264:"aud"`
}

func (s X264Settings) ForEach(fn func(key, value string) error) error {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("x264")
		if tag != "" {
			if err := fn(tag, v.Field(i).String()); err != nil {
				return err
			}
		}
	}
	return nil
}

var DefaultX264Settings = X264Settings{
	Preset:        "medium",
	Tune:          "film",
	Refs:          "6",
	Profile:       "high",
	Level:         "auto",
	Qmin:          "18",
	Qmax:          "28",
	BFrames:       "3",
	BAdapt:        "1",
	NGOP:          "250",
	NGOPMin:       "25",
	Scenecut:      "40",
	InfraRefresh:  "0",
	LookAhead:     "40",
	SlicedThreads: "0",
	SyncLookAhead: "1",
	ForceIDR:      "0",
	AQMode:        "1",
	AQStrength:    "1.0",
	MBTree:        "1",
	Bitrate:       "4000",
	VBVMaxBitrate: "5000",
	VBVBuffer:     "8000",
	RateTol:       "1",
	Threads:       "0",
	AnnexB:        "1",
	Aud:           "0",
}

var LowBandwidthX264Settings = X264Settings{
	Preset:        "veryfast",
	Tune:          "fastdecode",
	Refs:          "2",
	Profile:       "baseline",
	Level:         "4.1",
	Qmin:          "23",
	Qmax:          "35",
	BFrames:       "0",
	BAdapt:        "0",
	NGOP:          "60",
	NGOPMin:       "30",
	Scenecut:      "30",
	InfraRefresh:  "0",
	LookAhead:     "20",
	SlicedThreads: "1",
	SyncLookAhead: "0",
	ForceIDR:      "0",
	AQMode:        "0",
	AQStrength:    "1.2",
	MBTree:        "0",
	Bitrate:       "1500",
	VBVMaxBitrate: "1800",
	VBVBuffer:     "3000",
	RateTol:       "0.25",
	Threads:       "0",
	AnnexB:        "1",
	Aud:           "0",
}

var LowLatencyX264Settings = X264Settings{
	Preset:        "ultrafast",
	Tune:          "zerolatency",
	Refs:          "1",
	Profile:       "baseline",
	Level:         "4.1",
	Qmin:          "20",
	Qmax:          "32",
	BFrames:       "0",
	BAdapt:        "0",
	NGOP:          "30",
	NGOPMin:       "15",
	Scenecut:      "0",
	InfraRefresh:  "1",
	LookAhead:     "10",
	SlicedThreads: "1",
	SyncLookAhead: "0",
	ForceIDR:      "1",
	AQMode:        "0",
	AQStrength:    "0",
	MBTree:        "0",
	Bitrate:       "2500",
	VBVMaxBitrate: "3000",
	VBVBuffer:     "5000",
	RateTol:       "0.5",
	Threads:       "0",
	AnnexB:        "1",
	Aud:           "1",
}

var HighQualityX264Settings = X264Settings{
	Preset:        "slow",
	Tune:          "film",
	Refs:          "8",
	Profile:       "high",
	Level:         "5.1",
	Qmin:          "15",
	Qmax:          "24",
	BFrames:       "5",
	BAdapt:        "2",
	NGOP:          "250",
	NGOPMin:       "30",
	Scenecut:      "80",
	InfraRefresh:  "0",
	LookAhead:     "60",
	SlicedThreads: "0",
	SyncLookAhead: "1",
	ForceIDR:      "0",
	AQMode:        "0",
	AQStrength:    "1.3",
	MBTree:        "1",
	Bitrate:       "15000",
	VBVMaxBitrate: "20000",
	VBVBuffer:     "30000",
	RateTol:       "2.0",
	Threads:       "0",
	AnnexB:        "1",
	Aud:           "0",
}

func WithX264DefaultOptions(encoder *Encoder) error {
	encoder.encoderSettings = DefaultX264Settings
	return encoder.encoderSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264HighQualityOptions(encoder *Encoder) error {
	encoder.encoderSettings = HighQualityX264Settings
	return encoder.encoderSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264LowLatencyOptions(encoder *Encoder) error {
	encoder.encoderSettings = LowLatencyX264Settings
	return encoder.encoderSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264LowBandwidthOptions(encoder *Encoder) error {
	encoder.encoderSettings = LowBandwidthX264Settings
	return encoder.encoderSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func withVideoSetEncoderContextParameters(filter *Filter) EncoderOption {
	return func(encoder *Encoder) error {
		encoder.encoderContext.SetHeight(filter.sinkContext.Height())
		encoder.encoderContext.SetWidth(filter.sinkContext.Width())
		encoder.encoderContext.SetTimeBase(filter.sinkContext.TimeBase())
		encoder.encoderContext.SetPixelFormat(filter.sinkContext.PixelFormat())
		encoder.encoderContext.SetFramerate(filter.sinkContext.FrameRate())
		return nil
	}
}

func withAudioSetEncoderContextParameters(filter *Filter) EncoderOption {
	return func(encoder *Encoder) error {
		encoder.encoderContext.SetTimeBase(filter.sinkContext.TimeBase())
		encoder.encoderContext.SetSampleRate(filter.sinkContext.SampleRate())
		encoder.encoderContext.SetSampleFormat(filter.sinkContext.SampleFormat())
		encoder.encoderContext.SetChannelLayout(filter.sinkContext.ChannelLayout())
		encoder.encoderContext.SetStrictStdCompliance(-2)
		return nil
	}
}

func WithEncoderBufferSize(size int) EncoderOption {
	return func(encoder *Encoder) error {
		encoder.buffer = buffer.CreateChannelBuffer(encoder.ctx, size, internal.CreatePacketPool())
		return nil
	}
}
