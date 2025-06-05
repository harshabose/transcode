package transcode

import (
	"reflect"
	"strconv"
	"strings"
)

type X264AdvancedOptions struct {
	// PRIMARY OPTIONS
	Bitrate          string `x264-opts:"bitrate"`
	VBVMaxBitrate    string `x264-opts:"vbv-maxrate"`
	VBVBuffer        string `x264-opts:"vbv-bufsize"`
	RateTolerance    string `x264-opts:"ratetol"`
	MaxGOP           string `x264-opts:"keyint"`
	MinGOP           string `x264-opts:"min-keyint"`
	MaxQP            string `x264-opts:"qpmax"`
	MinQP            string `x264-opts:"qpmin"`
	MaxQPStep        string `x264-opts:"qpstep"`
	IntraRefresh     string `x264-opts:"intra-refresh"`
	ConstrainedIntra string `x264-opts:"constrained-intra"`

	// SECONDARY OPTIONS; SOME OF THEM ARE ALREADY SET BY PRESET, PROFILE AND TUNE
	SceneCut    string `x264-opts:"scenecut"`
	BFrames     string `x264-opts:"bframes"`
	BAdapt      string `x264-opts:"b-adapt"`
	Refs        string `x264-opts:"ref"`
	RCLookAhead string `x264-opts:"rc-lookahead"`
	AQMode      string `x264-opts:"aq-mode"`
	NalHrd      string `x264-opts:"nal-hrd"`
}

func (o *X264AdvancedOptions) ForEach(f func(key, value string) error) error {
	t := reflect.TypeOf(*o)
	v := reflect.ValueOf(*o)

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
		if err := f("x264opts", x264optsValue); err != nil {
			return err
		}
	}

	return nil
}

func (o *X264AdvancedOptions) UpdateBitrate(bps int64) error {
	kbps := bps / 1000

	// Core bitrate settings (strict CBR)
	o.Bitrate = strconv.FormatInt(kbps, 10)
	o.VBVMaxBitrate = strconv.FormatInt(kbps, 10) // Same as bitrate for CBR

	// VBV buffer: 0.5 seconds for low latency
	// Formula: buffer_kb = (bitrate_kbps * buffer_duration_seconds)
	// Minimum of 100 kb might be needed. TODO: do more research
	bufferKb := max(kbps/2, 100) // 0.5 seconds = 1/2 second
	o.VBVBuffer = strconv.FormatInt(bufferKb, 10)

	return nil
}

func (o *X264AdvancedOptions) GetCurrentBitrate() (int64, error) {
	kbps, err := strconv.ParseInt(o.Bitrate, 10, 64)
	if err != nil {
		return 0, err
	}
	return kbps * 1000, nil // Convert kbps to bps
}

type X264Options struct {
	*X264AdvancedOptions
	// PRECOMPILED OPTIONS
	Profile string `x264:"profile"`
	Level   string `x264:"level"`
	Preset  string `x264:"preset"`
	Tune    string `x264:"tune"`
}

func (o *X264Options) ForEach(f func(key, value string) error) error {
	t := reflect.TypeOf(*o)
	v := reflect.ValueOf(*o)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("x264")
		if tag != "" {
			if err := f(tag, v.Field(i).String()); err != nil {
				return err
			}
		}
	}

	return o.X264AdvancedOptions.ForEach(f)
}

func (o *X264Options) UpdateBitrate(bps int64) error {
	return o.X264AdvancedOptions.UpdateBitrate(bps)
}

// TODO: WARN: MAKING THIS A POINTER VARIABLE WILL MAKE ALL TRACKS WHICH USE THIS SETTINGS TO SHARE BITRATE

var LowLatencyBitrateControlled = &X264Options{
	Profile: "baseline",
	Level:   "3.1",
	Preset:  "ultrafast",
	Tune:    "zerolatency",

	X264AdvancedOptions: &X264AdvancedOptions{
		Bitrate:       "500", // 800kbps
		VBVMaxBitrate: "500",
		VBVBuffer:     "250",
		RateTolerance: "1", // 1% rate tolerance
		MaxGOP:        "25",
		MinGOP:        "13",
		// MaxQP:            "80",
		// MinQP:            "24",
		// MaxQPStep:        "80",
		IntraRefresh:     "0",
		ConstrainedIntra: "0",
		SceneCut:         "0",
		BFrames:          "0",
		BAdapt:           "0",
		Refs:             "1",
		RCLookAhead:      "0",
		AQMode:           "1", // Not sure; do more research
		NalHrd:           "cbr",
	},
}
