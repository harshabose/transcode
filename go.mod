module github.com/harshabose/simple_webrtc_comm/transcode

go 1.23.3

require (
	github.com/asticode/go-astiav v0.33.1
	github.com/harshabose/tools/buffer v0.0.0
	github.com/pion/rtp v1.8.11
	github.com/pion/webrtc/v4 v4.0.10
	github.com/asticode/go-astikit v0.52.0 // indirect
    github.com/pion/randutil v0.1.0 // indirect
)

replace github.com/harshabose/tools/buffer => ../tools/buffer
