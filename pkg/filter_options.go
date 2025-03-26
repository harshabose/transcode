package transcode

import (
	"fmt"
	"strings"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
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

func WithFilterBufferSize(size int) FilterOption {
	return func(filter *Filter) error {
		filter.buffer = buffer.CreateChannelBuffer(filter.ctx, size, internal.CreateFramePool())
		return nil
	}
}

func withVideoSetFilterContextParameters(decoder *Decoder) func(*Filter) error {
	return func(filter *Filter) error {
		filter.srcContextParams.SetHeight(decoder.decoderContext.Height())
		filter.srcContextParams.SetPixelFormat(decoder.decoderContext.PixelFormat())
		filter.srcContextParams.SetSampleAspectRatio(decoder.decoderContext.SampleAspectRatio())
		filter.srcContextParams.SetTimeBase(decoder.decoderContext.TimeBase())
		filter.srcContextParams.SetWidth(decoder.decoderContext.Width())

		filter.srcContextParams.SetColorSpace(decoder.decoderContext.ColorSpace())
		filter.srcContextParams.SetColorRange(decoder.decoderContext.ColorRange())

		return nil
	}
}

func WithVideoScaleFilterContent(width, height uint16) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("scale=%d:%d,", width, height)
		return nil
	}
}

func WithVideoPixelFormatFilterContent(pixelFormat astiav.PixelFormat) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("format=pix_fmts=%s,", pixelFormat)
		return nil
	}
}

func WithVideoFPSFilterContent(fps uint8) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("fps=%d,", fps)
		return nil
	}
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func withAudioSetFilterContextParameters(decoder *Decoder) func(*Filter) error {
	return func(filter *Filter) error {
		// Print parameter values before setting them
		fmt.Println("Setting filter parameters with values:")
		fmt.Printf("  Channel Layout: %v\n", decoder.decoderContext.ChannelLayout())
		fmt.Printf("  Sample Format: %v\n", decoder.decoderContext.SampleFormat())
		fmt.Printf("  Sample Rate: %v\n", decoder.decoderContext.SampleRate())
		fmt.Printf("  Time Base: %v\n", decoder.decoderContext.TimeBase())

		// Set the parameters
		filter.srcContextParams.SetChannelLayout(decoder.decoderContext.ChannelLayout())
		filter.srcContextParams.SetSampleFormat(decoder.decoderContext.SampleFormat())
		filter.srcContextParams.SetSampleRate(decoder.decoderContext.SampleRate())
		filter.srcContextParams.SetTimeBase(decoder.decoderContext.TimeBase())

		return nil
	}
}

func WithAudioSampleFormatChannelLayoutFilter(sampleFormat astiav.SampleFormat, channelLayout astiav.ChannelLayout) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("aformat=sample_fmts=%s:channel_layouts=%s", sampleFormat.String(), channelLayout.String()) + ","
		return nil
	}
}

func WithAudioSampleRateFilter(samplerate uint32) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("aresample=%d,", samplerate)
		return nil
	}
}

func WithAudioSamplesPerFrameContent(nsamples uint16) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("asetnsamples=%d,", nsamples)
		return nil
	}
}

func WithAudioCompressionContent(threshold int, ratio int, attack float64, release float64) FilterOption {
	return func(filter *Filter) error {
		// NOTE: DYNAMIC RANGE COMPRESSION TO HANDLE SUDDEN VOLUME CHANGES
		// Possible values 'acompressor=threshold=-12dB:ratio=2:attack=0.05:release=0.2" // MOST POPULAR VALUES
		filter.content += fmt.Sprintf("acompressor=threshold=%ddB:ratio=%d:attack=%.2f:release=%.2f,",
			threshold, ratio, attack, release)
		return nil
	}
}

func WithAudioHighPassContent(frequency int) FilterOption {
	return func(filter *Filter) error {
		// NOTE: HIGH-PASS FILTER TO REMOVE WIND NOISE AND TURBULENCE
		// NOTE: 120HZ CUTOFF MIGHT PRESERVE VOICE WHILE REMOVING LOW RUMBLE; BUT MORE TESTING IS NEEDED
		filter.content += fmt.Sprintf("highpass=f=%d,", frequency)
		return nil
	}
}

func WithAudioNotchFilterContent(id string, frequency float32, qFactor float32) FilterOption {
	return func(filter *Filter) error {
		filter.content += fmt.Sprintf("bandreject@%s=frequency=%.2f:width_type=q:width=%.2f", id, frequency, qFactor)
		return nil
	}
}

func WithAudioNotchHarmonicsFilterContent(id string, fundamental float32, harmonics uint8, qFactor float32) FilterOption {
	return func(filter *Filter) error {
		var filters = make([]string, 0)

		for i := uint8(0); i < harmonics; i++ {
			harmonic := fundamental * float32(i+1)
			filters = append(filters, fmt.Sprintf("bandreject@%s%d=frequency=%.2f:width_type=q:width=%.2f", id, i, harmonic, qFactor))
		}

		filter.content += strings.Join(filters, ",") + ","
		return nil
	}
}

// WARN: DO NOT USE FOR NOW
func WithAudioNeuralNetworkDenoiserContent(model string) FilterOption {
	return func(filter *Filter) error {
		// NOTE: A RECURRENT NEURAL NETWORK MIGHT BE THE BEST SOLUTION HERE BUT I AM NOT SURE HOW TO BUILD IT
		filter.content += fmt.Sprintf("arnndn=m=%s,", model)
		return nil
	}
}

func WithAudioEqualiserContent(frequency int, width int, gain int) FilterOption {
	return func(filter *Filter) error {
		// NOTE: EQUALISER CAN BE USED TO ENHANCE SPEECH BANDWIDTH (300 - 3kHz). MORE RESEARCH NEEDS TO DONE
		filter.content += fmt.Sprintf("equalizer=f=%d:t=h:width=%d:g=%d,",
			frequency, width, gain)
		return nil
	}
}

func WithAudioSilenceGateContent(threshold int, range_ int, attack float64, release float64) FilterOption {
	return func(filter *Filter) error {
		// NOTE: IF EVERYTHING WORKS, WE SHOULD HAVE LIGHT NOISE WHICH CAN BE CONSIDERED AS SILENCE. THIS GATE REMOVES SILENCE
		// NOTE: POSSIBLE VALUES 'agate=threshold=-30dB:range=-30dB:attack=0.01:release=0.1" // MOST POPULAR; MORE TESTING IS NEEDED
		filter.content += fmt.Sprintf("agate=threshold=%ddB:range=%ddB:attack=%.2f:release=%.2f,",
			threshold, range_, attack, release)
		return nil
	}
}

func WithAudioLoudnessNormaliseContent(intensity int, truePeak float64, range_ int) FilterOption {
	return func(filter *Filter) error {
		// NOTE: NORMALISES THE FINAL AUDIO. MUST BE CALLED AT THE END
		// NOTE: POSSIBLE VALUES "loudnorm=I=-16:TP=-1.5:LRA=11" // MOST POPULAR
		filter.content += fmt.Sprintf("loudnorm=I=%d:TP=%.1f:LRA=%d",
			intensity, truePeak, range_)
		return nil
	}
}

// WARN: DO NOT USE FOR NOW
func WithAudioNoiseReductionContent(strength int) FilterOption {
	return func(filter *Filter) error {
		// NOTE: anlmdn IS A NOISE REDUCTION FILTER. THIS MIGHT EFFECT THE QUALITY SIGNIFICANTLY - USE CAREFULLY
		filter.content += fmt.Sprintf("anlmdn=s=%d,", strength)
		return nil
	}
}
