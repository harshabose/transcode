module github.com/harshabose/simple_webrtc_comm/transcode

go 1.23.3

require (
	github.com/aler9/gomavlib v1.3.0
	github.com/asticode/go-astiav v0.33.1
	github.com/harshabose/tools/buffer v0.0.0
	github.com/pion/rtp v1.8.11
)

require (
	github.com/asticode/go-astikit v0.52.0 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07 // indirect
	golang.org/x/sys v0.1.0 // indirect
)

replace github.com/harshabose/tools/buffer => ../tools/buffer
