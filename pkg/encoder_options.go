package transcode

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type (
	EncoderOption = func(encoder *Encoder) error
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
	RateControl   string `x264:"rc"`
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
		Bitrate:       "2500",
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

var WebRTCOptimisedX264Settings = X264OpenSettings{
	X264Opts: X264Opts{
		// RateControl:   "cbr",
		Bitrate:       "800", // Keep your current target
		VBVMaxBitrate: "900", // Same as target!
		VBVBuffer:     "300", // 2500/30fps ≈ 83 kbits (single frame)
		RateTol:       "0.1", // More tolerance
		SyncLookAhead: "0",   // Already correct
		AnnexB:        "1",   // Already correct
	},
	LookAhead:     "0",   // Critical fix!
	Qmin:          "26",  // Wider range
	Qmax:          "42",  // Much wider range
	Level:         "3.1", // Better compatibility
	Preset:        "ultrafast",
	Tune:          "zerolatency",
	Refs:          "1",
	Profile:       "baseline",
	BFrames:       "0",
	BAdapt:        "0",
	NGOP:          "50",
	NGOPMin:       "25",
	Scenecut:      "0",
	InfraRefresh:  "1",
	SlicedThreads: "1",
	ForceIDR:      "1",
	AQMode:        "1",
	AQStrength:    "0.5",
	MBTree:        "0",

	Threads: "0",
	Aud:     "1",
}

func WithX264DefaultOptions(encoder *Encoder) error {
	encoder.codecSettings = DefaultX264Settings

	return encoder.codecSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264HighQualityOptions(encoder *Encoder) error {
	encoder.codecSettings = HighQualityX264Settings

	return encoder.codecSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264LowLatencyOptions(encoder *Encoder) error {
	encoder.codecSettings = LowLatencyX264Settings

	return encoder.codecSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithWebRTCOptimisedOptions(encoder *Encoder) error {
	encoder.codecSettings = WebRTCOptimisedX264Settings

	return encoder.codecSettings.ForEach(func(key, value string) error {
		fmt.Printf("setting key (%s): value(%s)\n", key, value)
		return encoder.codecFlags.Set(key, value, 0)
	})
}

func WithX264LowBandwidthOptions(encoder *Encoder) error {
	encoder.codecSettings = LowBandwidthX264Settings

	return encoder.codecSettings.ForEach(func(key, value string) error {
		return encoder.codecFlags.Set(key, value, 0)
	})
}

//
//
// func WithDefaultVP8Options(encoder *VP8Encoder) error {
// 	encoder.codecSettings = DefaultVP8Settings
//
// 	return encoder.codecSettings.ForEach(func(key, value string) error {
// 		if value == "" {
// 			return nil
// 		}
// 		return encoder.codecFlags.Set(key, value, 0)
// 	})
// }
//
// func withVideoSetEncoderParameters(filter *Filter) EncoderOption {
// 	return func(encoder *VP8Encoder) error {
// 		withVideoSetEncoderContextParameter(filter, encoder.encoderContext)
// 		return nil
// 	}
// }
//
// func withAudioSetEncoderParameters(filter *Filter) EncoderOption {
// 	return func(encoder *VP8Encoder) error {
// 		withAudioSetEncoderContextParameters(filter, encoder.encoderContext)
// 		return nil
// 	}
// }

func withAudioSetEncoderContextParameters(filter *Filter, eCtx *astiav.CodecContext) {
	eCtx.SetTimeBase(filter.sinkContext.TimeBase())
	eCtx.SetSampleRate(filter.sinkContext.SampleRate())
	eCtx.SetSampleFormat(filter.sinkContext.SampleFormat())
	eCtx.SetChannelLayout(filter.sinkContext.ChannelLayout())
	eCtx.SetStrictStdCompliance(-2)
}

func withVideoSetEncoderContextParameter(filter *Filter, eCtx *astiav.CodecContext) {
	eCtx.SetHeight(filter.sinkContext.Height())
	eCtx.SetWidth(filter.sinkContext.Width())
	eCtx.SetTimeBase(filter.sinkContext.TimeBase())
	eCtx.SetPixelFormat(filter.sinkContext.PixelFormat())
	eCtx.SetFramerate(filter.sinkContext.FrameRate())
}

func WithEncoderBufferSize(size int) EncoderOption {
	return func(encoder *Encoder) error {
		encoder.buffer = buffer.CreateChannelBuffer(encoder.ctx, size, internal.CreatePacketPool())
		return nil
	}
}

type VP8Settings struct {
	Deadline string `vp8:"deadline"` // Real-time encoding
	Bitrate  string `vp8:"b"`        // Target bitrate
	MinRate  string `vp8:"minrate"`  // Minimum bitrate
	MaxRate  string `vp8:"maxrate"`  // Maximum bitrate
	BufSize  string `vp8:"bufsize"`  // Buffer size
	CRF      string `vp8:"crf"`      // Quality setting
	CPUUsed  string `vp8:"cpu-used"` // Speed preset
}

var DefaultVP8Settings = VP8Settings{
	Deadline: "1",     // Real-time
	Bitrate:  "2500k", // 2.5 Mbps
	MinRate:  "2000k", // Min 2 Mbps
	MaxRate:  "3000k", // Max 3 Mbps
	BufSize:  "500k",  // 500kb buffer
	CRF:      "10",    // Good quality
	CPUUsed:  "8",     // Fastest
}

func (s VP8Settings) ForEach(fn func(key, value string) error) error {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("vp8")
		if tag != "" {
			if err := fn(tag, v.Field(i).String()); err != nil {
				return err
			}
		}
	}

	return nil
}
