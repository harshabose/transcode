package pkg

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

type Config struct {
	Source Name
	Sink   Name
}

const (
	videoBufferFilterName     Name = "buffer"
	videoBufferSinkFilterName Name = "buffersink"
)

var (
	VideoFilters = &Config{
		Source: videoBufferFilterName,
		Sink:   videoBufferSinkFilterName,
	}
)

func VideoSetFilterContextParameters(codecContext *astiav.CodecContext) func(*Filter) error {
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
