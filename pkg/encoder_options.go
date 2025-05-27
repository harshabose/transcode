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
	EncoderOption = func(encoder Encoder) error
)

type codecSettings interface {
	ForEach(func(string, string) error) error
}

type X264Opts struct {
	Bitrate       string `x264-opts:"bitrate"`
	VBVMaxBitrate string `x264-opts:"vbv-maxrate"`
	VBVBuffer     string `x264-opts:"vbv-bufsize"`
	RateTol       string `x264-opts:"ratetol"`
	SyncLookAhead string `x264-opts:"sync-lookahead"`
	AnnexB        string `x264-opts:"annexb"`
}

func (x264 *X264Opts) ForEach(fn func(string, string) error) error {
	t := reflect.TypeOf(*x264)
	v := reflect.ValueOf(*x264)

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

func (x264 *X264Opts) UpdateBitrate(bps int64) error {
	x264.Bitrate = fmt.Sprintf("%d", bps/1000)
	x264.VBVMaxBitrate = fmt.Sprintf("%d", (bps/1000)+200)
	x264.VBVBuffer = fmt.Sprintf("%d", bps/2000)

	return nil
}

type X264OpenSettings struct {
	*X264Opts
	// RateControl   string `x264:"rc"`            // not sure; fuck
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
	IntraRefresh  string `x264:"intra-refresh"` // exists
	LookAhead     string `x264:"rc-lookahead"`  // exists
	SlicedThreads string `x264:"slice"`         // exists
	ForceIDR      string `x264:"force-idr"`     // exists
	AQMode        string `x264:"aq-mode"`       // exists
	AQStrength    string `x264:"aq-strength"`   // exists
	MBTree        string `x264:"mbtree"`        // exists
	Threads       string `x264:"threads"`       // exists
	Aud           string `x264:"aud"`           // exists
}

func (s *X264OpenSettings) ForEach(fn func(key, value string) error) error {
	t := reflect.TypeOf(*s)
	v := reflect.ValueOf(*s)

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

func (s *X264OpenSettings) UpdateBitrate(bps int64) error {
	return s.X264Opts.UpdateBitrate(bps)
}

var DefaultX264Settings = X264OpenSettings{
	X264Opts: &X264Opts{
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
	IntraRefresh:  "0",
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
	X264Opts: &X264Opts{
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
	IntraRefresh:  "0",
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
	X264Opts: &X264Opts{
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
	IntraRefresh:  "1",
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
	X264Opts: &X264Opts{
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
	IntraRefresh:  "0",
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
	X264Opts: &X264Opts{
		Bitrate:       "800", // Keep your current target
		VBVMaxBitrate: "900", // Same as target!
		VBVBuffer:     "300", // 2500/30fps ≈ 83 kbits (single frame)
		RateTol:       "0.1", // More tolerance
		AnnexB:        "1",   // Already correct
	},
	Qmin:          "26",  // Wider range
	Qmax:          "42",  // Much wider range
	Level:         "3.1", // Better compatibility
	Preset:        "ultrafast",
	Tune:          "zerolatency",
	Profile:       "baseline",
	NGOP:          "50",
	NGOPMin:       "25",
	IntraRefresh:  "1",
	SlicedThreads: "1", // TODO: CHECK THIS
	// ForceIDR:      "1", // TODO: CHECK THIS; MIGHT BE IN CONFLICT WITH IntraRefresh
	AQMode:     "1", // RE-ENABLED AS zerolatency disables this
	AQStrength: "0.5",

	Threads: "0",
	Aud:     "1",
}

func WithX264DefaultOptions(encoder Encoder) error {
	return WithCodecSettings(&DefaultX264Settings)(encoder)
}

func WithX264HighQualityOptions(encoder Encoder) error {
	return WithCodecSettings(&HighQualityX264Settings)(encoder)
}

func WithX264LowLatencyOptions(encoder Encoder) error {
	return WithCodecSettings(&LowLatencyX264Settings)(encoder)
}

func WithWebRTCOptimisedOptions(encoder Encoder) error {
	return WithCodecSettings(&WebRTCOptimisedX264Settings)(encoder)
}

func WithCodecSettings(settings codecSettings) EncoderOption {
	return func(encoder Encoder) error {
		s, ok := encoder.(CanSetEncoderCodecSettings)
		if !ok {
			return ErrorInterfaceMismatch
		}

		return s.SetEncoderCodecSettings(settings)
	}
}

func WithX264LowBandwidthOptions(encoder Encoder) error {
	return WithCodecSettings(&LowBandwidthX264Settings)(encoder)
}

func withAudioSetEncoderContextParameters(filter CanDescribeMediaAudioFrame, eCtx *astiav.CodecContext) {
	eCtx.SetTimeBase(filter.TimeBase())
	eCtx.SetSampleRate(filter.SampleRate())
	eCtx.SetSampleFormat(filter.SampleFormat())
	eCtx.SetChannelLayout(filter.ChannelLayout())
	eCtx.SetStrictStdCompliance(-2)
}

func withVideoSetEncoderContextParameter(filter CanDescribeMediaVideoFrame, eCtx *astiav.CodecContext) {
	eCtx.SetHeight(filter.Height())
	eCtx.SetWidth(filter.Width())
	eCtx.SetTimeBase(filter.TimeBase())
	eCtx.SetPixelFormat(filter.PixelFormat())
	eCtx.SetFramerate(filter.FrameRate())
}

func WithEncoderBufferSize(size int) EncoderOption {
	return func(encoder Encoder) error {
		s, ok := encoder.(CanSetBuffer[astiav.Packet])
		if !ok {
			return ErrorInterfaceMismatch
		}
		s.SetBuffer(buffer.CreateChannelBuffer(encoder.Ctx(), size, internal.CreatePacketPool()))
		return nil
	}
}

//
// type VP8Settings struct {
// 	Deadline string `vp8:"deadline"` // Real-time encoding
// 	Bitrate  string `vp8:"b"`        // Target bitrate
// 	MinRate  string `vp8:"minrate"`  // Minimum bitrate
// 	MaxRate  string `vp8:"maxrate"`  // Maximum bitrate
// 	BufSize  string `vp8:"bufsize"`  // Buffer size
// 	CRF      string `vp8:"crf"`      // Quality setting
// 	CPUUsed  string `vp8:"cpu-used"` // Speed preset
// }
//
// var DefaultVP8Settings = VP8Settings{
// 	Deadline: "1",     // Real-time
// 	Bitrate:  "2500k", // 2.5 Mbps
// 	MinRate:  "2000k", // Min 2 Mbps
// 	MaxRate:  "3000k", // Max 3 Mbps
// 	BufSize:  "500k",  // 500kb buffer
// 	CRF:      "10",    // Good quality
// 	CPUUsed:  "8",     // Fastest
// }
//
// func (s VP8Settings) ForEach(fn func(key, value string) error) error {
// 	t := reflect.TypeOf(s)
// 	v := reflect.ValueOf(s)
//
// 	for i := 0; i < t.NumField(); i++ {
// 		field := t.Field(i)
// 		tag := field.Tag.Get("vp8")
// 		if tag != "" {
// 			if err := fn(tag, v.Field(i).String()); err != nil {
// 				return err
// 			}
// 		}
// 	}
//
// 	return nil
// }
