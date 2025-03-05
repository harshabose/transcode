package transcode

import (
	"reflect"
	"strings"

	buffer "github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type (
	EncoderOption = func(*Encoder) error
)

type codecSettings interface {
	ForEach(func(string, string) error) error
}

type X264Opts struct {
	// RateControl   string `x264-opts:"rate-control"`
	Bitrate       string `x264-opts:"bitrate"`
	VBVMaxBitrate string `x264-opts:"vbv-maxrate"`
	VBVBuffer     string `x264-opts:"vbv-bufsize"`
	RateTol       string `x264-opts:"ratetol"`
	SyncLookAhead string `x264-opts:"sync-lookahead"`
	AnnexB        string `x264-opts:"annexb"`
}

func (x264 X264Opts) ForEach(fn func(string, string) error) error {
	t := reflect.TypeOf(x264)
	v := reflect.ValueOf(x264)

	// Build a single x264opts string
	var optParts []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("x264-opts")
		if tag != "" && v.Field(i).String() != "" {
			optParts = append(optParts, tag+"="+v.Field(i).String())
		}
	}

	// Join all options with colons
	if len(optParts) > 0 {
		x264optsValue := strings.Join(optParts, ":")
		// Set as a single parameter
		if err := fn("x264opts", x264optsValue); err != nil {
			return err
		}
	}

	// Also apply any regular x264 parameters
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

type X264OpenSettings struct {
	X264Opts
	Preset        string `x264:"preset"`        // exists
	Tune          string `x264:"tune"`          // exists
	Refs          string `x264:"refs"`          // exists
	Profile       string `x264:"profile"`       // exists
	Level         string `x264:"level"`         // exists
	Qmin          string `x264:"qmin"`          // exists
	Qmax          string `x264:"qmax"`          // exists
	BFrames       string `x264:"bf"`            // exists
	BAdapt        string `x264:"b_strategy"`    // exists
	NGOP          string `x264:"g"`             // exists
	NGOPMin       string `x264:"keyint_min"`    // exists
	Scenecut      string `x264:"sc_threshold"`  // exists
	InfraRefresh  string `x264:"intra-refresh"` // exists
	LookAhead     string `x264:"rc-lookahead"`  // exists
	SlicedThreads string `x264:"slice"`         // exists
	ForceIDR      string `x264:"force-idr"`     // exists
	AQMode        string `x264:"aq-mode"`       // exists
	AQStrength    string `x264:"aq-strength"`   // exists
	MBTree        string `x264:"mbtree"`        // exists
	Threads       string `x264:"threads"`       // exists
	Aud           string `x264:"aud"`           // exists
}

func (s X264OpenSettings) ForEach(fn func(key, value string) error) error {
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

	return s.X264Opts.ForEach(fn)
}

var DefaultX264Settings = X264OpenSettings{
	X264Opts: X264Opts{
		// RateControl:   "abr",
		Bitrate:       "4000",
		VBVMaxBitrate: "5000",
		VBVBuffer:     "8000",
		RateTol:       "1",
		SyncLookAhead: "1",
		AnnexB:        "1",
	},
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
	ForceIDR:      "0",
	AQMode:        "1",
	AQStrength:    "1.0",
	MBTree:        "1",
	Threads:       "0",
	Aud:           "0",
}

var LowBandwidthX264Settings = X264OpenSettings{
	X264Opts: X264Opts{
		// RateControl:   "abr",
		Bitrate:       "1500",
		VBVMaxBitrate: "1800",
		VBVBuffer:     "3000",
		RateTol:       "0.25",
		SyncLookAhead: "0",
		AnnexB:        "1",
	},
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
	ForceIDR:      "0",
	AQMode:        "0",
	AQStrength:    "1.2",
	MBTree:        "0",
	Threads:       "0",
	Aud:           "0",
}

var LowLatencyX264Settings = X264OpenSettings{
	X264Opts: X264Opts{
		// RateControl:   "abr",
		Bitrate:       "5000",
		VBVMaxBitrate: "12000",
		VBVBuffer:     "20000",
		RateTol:       "0.5",
		SyncLookAhead: "0",
		AnnexB:        "1",
	},
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
	ForceIDR:      "1",
	AQMode:        "0",
	AQStrength:    "0",
	MBTree:        "0",

	Threads: "0",
	Aud:     "1",
}

var HighQualityX264Settings = X264OpenSettings{
	X264Opts: X264Opts{
		// RateControl:   "abr",
		Bitrate:       "15000",
		VBVMaxBitrate: "20000",
		VBVBuffer:     "30000",
		RateTol:       "2.0",
		SyncLookAhead: "1",
		AnnexB:        "1",
	},
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
	ForceIDR:      "0",
	AQMode:        "0",
	AQStrength:    "1.3",
	MBTree:        "1",

	Threads: "0",
	Aud:     "0",
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
