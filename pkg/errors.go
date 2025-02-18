package pkg

import "errors"

var (
	ErrorAllocateFormatContext = errors.New("error allocate format context")
	ErrorOpenInputContainer    = errors.New("error opening container")
	ErrorNoStreamFound         = errors.New("error no stream found")
	ErrorNoVideoStreamFound    = errors.New("no video stream found")

	ErrorNoCodecFound         = errors.New("error no codec found")
	ErrorAllocateCodecContext = errors.New("error allocating codec context")
	ErrorFillCodecContext     = errors.New("error filling the codec context")

	ErrorNoFilterName           = errors.New("error filter name does not exists")
	WarnNoFilterContent         = errors.New("content is empty. no filtering will be done")
	ErrorGraphParse             = errors.New("error parsing the filter graph")
	ErrorGraphConfigure         = errors.New("error configuring the filter graph")
	ErrorSrcContextSetParameter = errors.New("error while setting parameters to source context")
	ErrorSrcContextInitialise   = errors.New("error initialising the source context")
	ErrorAllocSrcContext        = errors.New("error setting source context")
	ErrorAllocSinkContext       = errors.New("error setting sink context")

	ErrorCodecNoSetting = errors.New("error no settings given")
)
