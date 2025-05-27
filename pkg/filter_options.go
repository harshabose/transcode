package transcode

import (
	"fmt"
	"strings"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
)

type (
	FilterOption func(Filter) error
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
	VideoFilters = FilterConfig{
		Source: videoBufferFilterName,
		Sink:   videoBufferSinkFilterName,
	}
	AudioFilters = FilterConfig{
		Source: audioBufferFilterName,
		Sink:   audioBufferSinkFilterName,
	}
)

func WithFilterBufferSize(size int) FilterOption {
	return func(filter Filter) error {
		s, ok := filter.(CanSetBuffer[astiav.Frame])
		if !ok {
			return ErrorInterfaceMismatch
		}
		s.SetBuffer(buffer.CreateChannelBuffer(filter.Ctx(), size, internal.CreateFramePool()))
		return nil
	}
}

func withVideoSetFilterContextParameters(decoder CanDescribeMediaVideoFrame) func(Filter) error {
	return func(filter Filter) error {
		canSetMediaVideoFrame, ok := filter.(CanSetMediaVideoFrame)
		if !ok {
			return ErrorInterfaceMismatch
		}

		canSetMediaVideoFrame.SetFrameRate(decoder)
		canSetMediaVideoFrame.SetHeight(decoder)
		canSetMediaVideoFrame.SetPixelFormat(decoder)
		canSetMediaVideoFrame.SetSampleAspectRatio(decoder)
		canSetMediaVideoFrame.SetTimeBase(decoder)
		canSetMediaVideoFrame.SetWidth(decoder)

		canSetMediaVideoFrame.SetColorSpace(decoder)
		canSetMediaVideoFrame.SetColorRange(decoder)

		return nil
	}
}

func WithVideoScaleFilterContent(width, height uint16) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("scale=%d:%d,", width, height))
		return nil
	}
}

func WithVideoPixelFormatFilterContent(pixelFormat astiav.PixelFormat) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}
		fmt.Println("pixel filter added:", pixelFormat.String())
		a.AddToFilterContent(fmt.Sprintf("format=pix_fmts=%s,", pixelFormat))
		return nil
	}
}

func WithVideoFPSFilterContent(fps uint8) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("fps=%d,", fps))
		return nil
	}
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func withAudioSetFilterContextParameters(decoder CanDescribeMediaAudioFrame) func(Filter) error {
	return func(filter Filter) error {
		canSetMediaAudioFrame, ok := filter.(CanSetMediaAudioFrame)
		if !ok {
			return ErrorInterfaceMismatch
		}
		canSetMediaAudioFrame.SetChannelLayout(decoder)
		canSetMediaAudioFrame.SetSampleFormat(decoder)
		canSetMediaAudioFrame.SetSampleRate(decoder)
		canSetMediaAudioFrame.SetTimeBase(decoder)

		return nil
	}
}

func WithAudioSampleFormatChannelLayoutFilter(sampleFormat astiav.SampleFormat, channelLayout astiav.ChannelLayout) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("aformat=sample_fmts=%s:channel_layouts=%s", sampleFormat.String(), channelLayout.String()) + ",")
		return nil
	}
}

func WithAudioSampleRateFilter(samplerate uint32) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("aresample=%d,", samplerate))
		return nil
	}
}

func WithAudioSamplesPerFrameContent(nsamples uint16) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("asetnsamples=%d,", nsamples))
		return nil
	}
}

func WithAudioCompressionContent(threshold int, ratio int, attack float64, release float64) FilterOption {
	return func(filter Filter) error {
		// NOTE: DYNAMIC RANGE COMPRESSION TO HANDLE SUDDEN VOLUME CHANGES
		// Possible values 'acompressor=threshold=-12dB:ratio=2:attack=0.05:release=0.2" // MOST POPULAR VALUES
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("acompressor=threshold=%ddB:ratio=%d:attack=%.2f:release=%.2f,",
			threshold, ratio, attack, release))
		return nil
	}
}

func WithAudioHighPassFilterContent(id string, frequency float32, order uint8) FilterOption {
	return func(filter Filter) error {
		// NOTE: HIGH-PASS FILTER TO REMOVE WIND NOISE AND TURBULENCE
		// NOTE: 120HZ CUTOFF MIGHT PRESERVE VOICE WHILE REMOVING LOW RUMBLE; BUT MORE TESTING IS NEEDED
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("highpass@%s=frequency=%.2f:poles=%d", id, frequency, order))
		return nil
	}
}

func WithAudioLowPassFilterContent(id string, frequency float32, order uint8) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("lowpass@%s=frequency=%.2f:poles=%d", id, frequency, order))
		return nil
	}
}

func WithAudioNotchFilterContent(id string, frequency float32, qFactor float32) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("bandreject@%s=frequency=%.2f:width_type=q:width=%.2f", id, frequency, qFactor))
		return nil
	}
}

func WithAudioNotchHarmonicsFilterContent(id string, fundamental float32, harmonics uint8, qFactor float32) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		var filters = make([]string, 0)

		for i := uint8(0); i < harmonics; i++ {
			harmonic := fundamental * float32(i+1)
			filters = append(filters, fmt.Sprintf("bandreject@%s%d=frequency=%.2f:width_type=q:width=%.2f", id, i, harmonic, qFactor))
		}

		a.AddToFilterContent(strings.Join(filters, ",") + ",")
		return nil
	}
}

func WithAudioEqualiserFilter(id string, frequency float32, width float32, gain float32) FilterOption {
	return func(filter Filter) error {
		// NOTE: EQUALISER CAN BE USED TO ENHANCE SPEECH BANDWIDTH (300 - 3kHz). MORE RESEARCH NEEDS TO DONE
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("equalizer@%s=frequency=%.2f:width_type=h:width=%.2f:gain=%.2f,", id, frequency, width, gain))
		return nil
	}
}

func WithAudioSilenceGateContent(threshold int, range_ int, attack float64, release float64) FilterOption {
	return func(filter Filter) error {
		// NOTE: IF EVERYTHING WORKS, WE SHOULD HAVE LIGHT NOISE WHICH CAN BE CONSIDERED AS SILENCE. THIS GATE REMOVES SILENCE
		// NOTE: POSSIBLE VALUES 'agate=threshold=-30dB:range=-30dB:attack=0.01:release=0.1" // MOST POPULAR; MORE TESTING IS NEEDED
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("agate=threshold=%ddB:range=%ddB:attack=%.2f:release=%.2f,",
			threshold, range_, attack, release))
		return nil
	}
}

func WithAudioLoudnessNormaliseContent(intensity int, truePeak float64, range_ int) FilterOption {
	return func(filter Filter) error {
		// NOTE: NORMALISES THE FINAL AUDIO. MUST BE CALLED AT THE END
		// NOTE: POSSIBLE VALUES "loudnorm=I=-16:TP=-1.5:LRA=11" // MOST POPULAR
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("loudnorm=I=%d:TP=%.1f:LRA=%d",
			intensity, truePeak, range_))
		return nil
	}
}

func WithFFTBroadBandNoiseFilter(id string, strength float32, rPatch float32, rSearch float32) FilterOption {
	return func(filter Filter) error {
		// TODO: NEEDS A UPDATOR TO CONTROL NOISE SAMPLING
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf(""))
		return nil
	}
}

func WithMeanBroadBandNoiseFilter(id string, strength float32, rPatch float32, rSearch float32) FilterOption {
	return func(filter Filter) error {
		a, ok := filter.(CanAddToFilterContent)
		if !ok {
			return ErrorInterfaceMismatch
		}

		a.AddToFilterContent(fmt.Sprintf("anlmdn@%s=strength=%.2f:patch=%.2f:research=%.2f", id, strength, rPatch, rSearch))
		return nil
	}
}
