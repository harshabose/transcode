package transcode

import (
	"fmt"

	"github.com/asticode/go-astiav"
)

type (
	FilterOption func(*Filter) error
	Name         string
)

func (f Name) String() string {
	return string(f)
}

type FilterConfig struct {
	Source Name
	Sink   Name
}

const (
	videoBufferFilterName     Name = "buffer"
	videoBufferSinkFilterName Name = "buffersink"
	audioBufferFilterName     Name = "abuffer"
	audioBufferSinkFilterName Name = "abuffersink"
)

var (
	VideoFilters = &FilterConfig{
		Source: videoBufferFilterName,
		Sink:   videoBufferSinkFilterName,
	}
	AudioFilters = &FilterConfig{
		Source: audioBufferFilterName,
		Sink:   audioBufferSinkFilterName,
	}
)

func withVideoSetFilterContextParameters(codecContext *astiav.CodecContext) func(*Filter) error {
	return func(filter *Filter) error {
		filter.srcContextParams.SetHeight(codecContext.Height())
		filter.srcContextParams.SetPixelFormat(codecContext.PixelFormat())
		filter.srcContextParams.SetSampleAspectRatio(codecContext.SampleAspectRatio())
		filter.srcContextParams.SetTimeBase(codecContext.TimeBase())
		filter.srcContextParams.SetWidth(codecContext.Width())
		return nil
	}
}

func WithDefaultVideoFilterContentOptions(filter *Filter) error {
	if err := videoScaleFilterContent(filter); err != nil {
		return err
	}
	if err := videoPixelFormatFilterContent(filter); err != nil {
		return err
	}
	if err := videoFPSFilterContent(filter); err != nil {
		return err
	}

	return nil
}

func videoScaleFilterContent(filter *Filter) error {
	filter.content += fmt.Sprintf("scale=%d:%d,", DefaultVideoWidth, DefaultVideoHeight)
	return nil
}

func videoPixelFormatFilterContent(filter *Filter) error {
	filter.content += fmt.Sprintf("format=pix_fmts=%s,", DefaultVideoPixFormat)
	return nil
}

func videoFPSFilterContent(filter *Filter) error {
	filter.content += fmt.Sprintf("fps=%d,", DefaultVideoFPS)
	return nil
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func withAudioSetFilterContextParameters(codecContext *astiav.CodecContext) func(*Filter) error {
	return func(filter *Filter) error {
		filter.srcContextParams.SetChannelLayout(codecContext.ChannelLayout())
		filter.srcContextParams.SetSampleFormat(codecContext.SampleFormat())
		filter.srcContextParams.SetSampleRate(codecContext.SampleRate())
		filter.srcContextParams.SetTimeBase(codecContext.TimeBase())

		return nil
	}
}

func WithDefaultAudioFilterContentOptions(filter *Filter) error {
	if err := audioSampleFormatChannelLayoutContent(filter); err != nil {
		return err
	}
	if err := audioSampleRateContent(filter); err != nil {
		return err
	}
	if err := audioFrameSizeContent(filter); err != nil {
		return err
	}
	return nil
}

func audioSampleFormatChannelLayoutContent(filter *Filter) error {
	filter.content += buildAudioFormatContent(DefaultAudioSampleFormat, DefaultAudioChannelLayout) + ","
	return nil
}

func buildAudioFormatContent(sampleFormat astiav.SampleFormat, channelLayout astiav.ChannelLayout) string {
	return fmt.Sprintf("aformat=sample_fmts=%s:channel_layouts=%s", sampleFormat.String(), channelLayout.String())
}

func audioSampleRateContent(filter *Filter) error {
	filter.content += fmt.Sprintf("aresample=%d,", DefaultAudioSampleRate)
	return nil
}

func audioFrameSizeContent(filter *Filter) error {
	filter.content += fmt.Sprintf("asetnsamples=%d,", DefaultAudioFrameSize)
	return nil
}

func audioCompressionContent(filter *Filter) error {
	// NOTE: DYNAMIC RANGE COMPRESSION TO HANDLE SUDDEN VOLUME CHANGES
	// Possible values 'acompressor=threshold=-12dB:ratio=2:attack=0.05:release=0.2" // MOST POPULAR VALUES
	filter.content += fmt.Sprintf("acompressor=threshold=%ddB:ratio=%d:attack=%d:release=%d,")
	return nil
}

func audioHighPassContent(filter *Filter) error {
	// NOTE: HIGH-PASS FILTER TO REMOVE WIND NOISE AND TURBULENCE
	// NOTE: 120HZ CUTOFF MIGHT PRESERVE VOICE WHILE REMOVING LOW RUMBLE; BUT MORE TESTING IS NEEDED
	filter.content += fmt.Sprintf("highpass=f=%d,")
	return nil
}

func audioNotchFilterContent(filter *Filter) error {
	// NOTE: NOTCH FILTER CAN BE USED TO TARGET SPECIFIC PROPELLER NOISE AND REMOVE THEM
	// NOTE: THIS MIGHT BE UNIQUE TO DRONE AND POWER LEVELS. I AM NOT SURE HOW TO USE IT TOO.
	filter.content += "afftfilt=real='re*cos(0)':imag='im*cos(0):win_size=1024:fixed=true',"
	return nil
}

// WARN: DO NOT USE FOR NOW
func audioNeuralNetworkDenoiserContent(filter *Filter) error {
	// NOTE: A RECURRENT NEURAL NETWORK MIGHT BE THE BEST SOLUTION HERE BUT I AM NOT SURE HOW TO BUILD IT
	filter.content += "arnndn=m=,"
	return nil
}

func audioEqualiser(filter *Filter) error {
	// NOTE: EQUALISER CAN BE USED TO ENHANCE SPEECH BANDWIDTH (300 - 3kHz). MORE RESEARCH NEEDS TO DONE
	filter.content += fmt.Sprintf("equalizer=f=%d:t=h:width=%d:g=%d,")

	return nil
}

func audioSilenceGateContent(filter *Filter) error {
	// NOTE: IF EVERYTHING WORKS, WE SHOULD HAVE LIGHT NOISE WHICH CAN BE CONSIDERED AS SILENCE. THIS GATE REMOVES SILENCE
	// NOTE: POSSIBLE VALUES 'agate=threshold=-30dB:range=-30dB:attack=0.01:release=0.1" // MOST POPULAR; MORE TESTING IS NEEDED
	filter.content += fmt.Sprintf("agate=threshold=%ddB:range=%ddB:attack=%d:release=%d,")
	return nil
}

func audioLoudnessNormaliseContent(filter *Filter) error {
	// NOTE: NORMALISES THE FINAL AUDIO. MUST BE CALLED AT THE END
	// NOTE: POSSIBLE VALUES "loudnorm=I=-16:TP=-1.5:LRA=11" // MOST POPULAR
	filter.content += fmt.Sprintf("loudnorm=I=%d:TP=%d:LRA=%d")
	return nil
}

// WARN: DO NOT USE FOR NOW
func audioNoiseReductionContent(filter *Filter) error {
	// NOTE: anlmdn IS A NOISE REDUCTION FILTER. THIS MIGHT EFFECT THE QUALITY SIGNIFICANTLY - USE CAREFULLY
	filter.content += fmt.Sprintf("anlmdn=s=%d,")
	return nil
}
