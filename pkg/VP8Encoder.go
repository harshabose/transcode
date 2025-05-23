package transcode

//
// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"math"
// 	"sync"
// 	"time"
//
// 	"github.com/asticode/go-astiav"
//
// 	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
// 	"github.com/harshabose/tools/buffer/pkg"
// )
//
// type VP8Encoder struct {
// 	buffer         buffer.BufferWithGenerator[astiav.Packet]
// 	filter         *Filter
// 	codec          *astiav.Codec
// 	codecFlags     *astiav.Dictionary
// 	copyCodecFlags *astiav.Dictionary
// 	codecSettings  codecSettings
// 	bandwidthChan  chan int64
// 	options        []EncoderOption
//
// 	encoderContext         *astiav.CodecContext
// 	fallbackEncoderContext *astiav.CodecContext
//
// 	ctx context.Context
// 	mux sync.Mutex
// }
//
// func NewVP8Encoder(ctx context.Context, filter *Filter, options ...EncoderOption) (*VP8Encoder, error) {
// 	encoder := &VP8Encoder{
// 		filter:     filter,
// 		codecFlags: astiav.NewDictionary(),
// 		ctx:        ctx,
// 	}
//
// 	if encoder.codec = astiav.FindEncoder(astiav.CodecIDVp8); encoder.codec == nil {
// 		return nil, errors.New("VP8 encoder not found")
// 	}
//
// 	encoderContext, err := createNewVP8Encoder(encoder.codec, filter)
// 	if err != nil {
// 		return nil, err
// 	}
// 	encoder.encoderContext = encoderContext
//
// 	for _, option := range options {
// 		if err := option(encoder); err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	if encoder.codecSettings == nil {
// 		fmt.Println("warn: no VP8 encoder settings were provided")
// 	}
//
// 	copyDict, err := copyDictionary(encoder.codecFlags)
// 	if err != nil {
// 		return nil, err
// 	}
// 	encoder.copyCodecFlags = copyDict
//
// 	if err := openVP8Encoder(encoder.encoderContext, encoder.codec, encoder.codecFlags); err != nil {
// 		return nil, err
// 	}
//
// 	if encoder.buffer == nil {
// 		encoder.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreatePacketPool())
// 	}
//
// 	return encoder, nil
// }
//
// func (e *VP8Encoder) Start() {
// 	go e.loop()
// }
//
// func (e *VP8Encoder) GetPacket() (*astiav.Packet, error) {
// 	ctx, cancel := context.WithTimeout(e.ctx, time.Second)
// 	defer cancel()
// 	return e.buffer.Pop(ctx)
// }
//
// func (e *VP8Encoder) WaitForPacket() chan *astiav.Packet {
// 	return e.buffer.GetChannel()
// }
//
// func (e *VP8Encoder) PutBack(packet *astiav.Packet) {
// 	e.buffer.PutBack(packet)
// }
//
// func (e *VP8Encoder) GetTimeBase() astiav.Rational {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
//
// 	if e.encoderContext != nil {
// 		return e.encoderContext.TimeBase()
// 	}
// 	if e.fallbackEncoderContext != nil {
// 		return e.fallbackEncoderContext.TimeBase()
// 	}
// 	return astiav.Rational{}
// }
//
// func (e *VP8Encoder) GetDuration() time.Duration {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
//
// 	if e.encoderContext != nil {
// 		return time.Duration(float64(time.Second) / e.encoderContext.Framerate().Float64())
// 	}
// 	if e.fallbackEncoderContext != nil {
// 		return time.Duration(float64(time.Second) / e.fallbackEncoderContext.Framerate().Float64())
// 	}
// 	return time.Second / 30
// }
//
// func (e *VP8Encoder) SetBitrateChannel(channel chan int64) {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
// 	e.bandwidthChan = channel
// }
//
// // Get current VP8 bitrate from encoder context
// func (e *VP8Encoder) getCurrentBitrate() (int64, error) {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
//
// 	if e.encoderContext != nil {
// 		return e.encoderContext.BitRate() / 1000, nil // Convert to kbps
// 	}
// 	return 0, errors.New("no encoder context available")
// }
//
// // Update VP8 bitrate (simpler than x264)
// func (e *VP8Encoder) updateBitrate(bitrate int64) error {
// 	start := time.Now()
//
// 	e.mux.Lock()
// 	current, err := e.getCurrentBitrate()
// 	if err != nil {
// 		e.mux.Unlock()
// 		fmt.Printf("error getting current bitrate; err: %s\n", err.Error())
// 		return err
// 	}
//
// 	// Same change logic as your x264 version
// 	change := math.Abs(float64(current)-float64(bitrate)) / math.Abs(float64(current))
//
// 	if change < 0.1 || change > 2.0 {
// 		e.mux.Unlock()
// 		fmt.Printf("change not appropriate; current: %d; new: %d; change:%f\n", current, bitrate, change)
// 		return nil
// 	}
//
// 	fmt.Printf("VP8 bitrate change approved; change: %f\n", change)
//
// 	// Set VP8 bitrate parameters
// 	if err := e.updateVP8Options(bitrate); err != nil {
// 		e.mux.Unlock()
// 		fmt.Printf("error while updating VP8 options; err: %s\n", err.Error())
// 		return err
// 	}
//
// 	e.mux.Unlock()
// 	if err := e.createNewEncoderContext(); err != nil {
// 		return err
// 	}
//
// 	duration := time.Since(start)
// 	fmt.Printf("🔄 VP8 Bitrate updated: %d → %d (%.1f%%) in %v\n",
// 		current, bitrate, change*100, duration)
//
// 	return nil
// }
//
// // Update VP8-specific options
// func (e *VP8Encoder) updateVP8Options(bitrate int64) error {
// 	// VP8 uses simpler parameter names
// 	paramsToUpdate := map[string]string{
// 		"deadline": "1",                                 // Real-time encoding
// 		"b:v":      fmt.Sprintf("%dk", bitrate),         // Target bitrate
// 		"minrate":  fmt.Sprintf("%dk", bitrate*80/100),  // Min bitrate (80% of target)
// 		"maxrate":  fmt.Sprintf("%dk", bitrate*120/100), // Max bitrate (120% of target)
// 		"bufsize":  fmt.Sprintf("%dk", bitrate/5),       // Buffer size
// 		"crf":      "10",                                // Good quality balance
// 		"cpu-used": "8",                                 // Fastest preset for real-time
// 	}
//
// 	for param, value := range paramsToUpdate {
// 		if err := e.copyCodecFlags.Set(param, value, 0); err != nil {
// 			return err
// 		}
// 	}
//
// 	return nil
// }
//
// // Rest of your encoder methods...
// func (e *VP8Encoder) createNewEncoderContext() error {
// 	e.mux.Lock()
// 	e.fallbackEncoderContext = e.encoderContext
// 	e.encoderContext = nil
//
// 	copyDict, err := copyDictionary(e.copyCodecFlags)
// 	if err != nil {
// 		e.mux.Unlock()
// 		return err
// 	}
//
// 	e.codecFlags.Free()
// 	e.codecFlags = copyDict
// 	e.mux.Unlock()
//
// 	encoderContext, err := createNewOpenVP8Encoder(e.codec, e.filter, e.codecFlags)
// 	if err != nil {
// 		e.mux.Lock()
// 		e.encoderContext = e.fallbackEncoderContext
// 		e.fallbackEncoderContext = nil
// 		e.mux.Unlock()
// 		fmt.Printf("New VP8 encoder creation failed, reverted: %v\n", err)
// 		return err
// 	}
//
// 	e.mux.Lock()
// 	oldFallback := e.fallbackEncoderContext
// 	e.encoderContext = encoderContext
// 	e.fallbackEncoderContext = nil
// 	e.mux.Unlock()
//
// 	if oldFallback != nil {
// 		oldFallback.Free()
// 		fmt.Printf("🧹 Cleaned up fallback VP8 encoder context\n")
// 	}
//
// 	return nil
// }
//
// func (e *VP8Encoder) pickContextAndProcess(frame *astiav.Frame) error {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
//
// 	if e.encoderContext != nil {
// 		return e.sendFrameAndPutPackets(e.encoderContext, frame)
// 	}
// 	if e.fallbackEncoderContext != nil {
// 		return e.sendFrameAndPutPackets(e.fallbackEncoderContext, frame)
// 	}
// 	return errors.New("invalid VP8 encoder context state")
// }
//
// func (e *VP8Encoder) sendFrameAndPutPackets(encoderContext *astiav.CodecContext, frame *astiav.Frame) error {
// 	defer e.filter.PutBack(frame)
//
// 	if err := encoderContext.SendFrame(frame); err != nil {
// 		return err
// 	}
//
// 	for {
// 		packet := e.buffer.Generate()
// 		if err := encoderContext.ReceivePacket(packet); err != nil {
// 			e.buffer.PutBack(packet)
// 			break
// 		}
// 		if err := e.pushPacket(packet); err != nil {
// 			e.buffer.PutBack(packet)
// 			continue
// 		}
// 	}
// 	return nil
// }
//
// func (e *VP8Encoder) pushPacket(packet *astiav.Packet) error {
// 	ctx, cancel := context.WithTimeout(e.ctx, time.Second)
// 	defer cancel()
// 	return e.buffer.Push(ctx, packet)
// }
//
// func (e *VP8Encoder) loop() {
// 	e.encoderContext.SetBitRate(2_000_000)
// 	fmt.Println("VP8 loop started")
// 	defer e.Close()
//
// 	for {
// 		select {
// 		case <-e.ctx.Done():
// 			return
// 		case bitrate := <-e.bandwidthChan:
// 			fmt.Println("bitrate recommended:", bitrate)
// 			// if err := e.updateBitrate(bitrate); err != nil {
// 			// 	fmt.Printf("error while updating VP8 bitrate; err: %s\n", err.Error())
// 			// }
// 		case frame := <-e.filter.WaitForFrame():
// 			if err := e.pickContextAndProcess(frame); err != nil {
// 				if !errors.Is(err, astiav.ErrEagain) {
// 					continue
// 				}
// 			}
// 		}
// 	}
// }
//
// func (e *VP8Encoder) Close() {
// 	e.mux.Lock()
// 	defer e.mux.Unlock()
//
// 	if e.encoderContext != nil {
// 		e.encoderContext.Free()
// 		e.encoderContext = nil
// 	}
// 	if e.fallbackEncoderContext != nil {
// 		e.fallbackEncoderContext.Free()
// 		e.fallbackEncoderContext = nil
// 	}
// }
//
// // Helper functions for VP8 encoder creation
// func createNewVP8Encoder(codec *astiav.Codec, filter *Filter) (*astiav.CodecContext, error) {
// 	encoderContext := astiav.AllocCodecContext(codec)
// 	if encoderContext == nil {
// 		return nil, ErrorAllocateCodecContext
// 	}
//
// 	// Set VP8-specific context parameters
// 	withVideoSetEncoderContextParameter(filter, encoderContext)
//
// 	return encoderContext, nil
// }
//
// func createNewOpenVP8Encoder(codec *astiav.Codec, filter *Filter, settings *astiav.Dictionary) (*astiav.CodecContext, error) {
// 	encoderContext, err := createNewVP8Encoder(codec, filter)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if err := openVP8Encoder(encoderContext, codec, settings); err != nil {
// 		encoderContext.Free()
// 		return nil, err
// 	}
//
// 	return encoderContext, nil
// }
//
// func openVP8Encoder(encoderContext *astiav.CodecContext, codec *astiav.Codec, settings *astiav.Dictionary) error {
// 	encoderContext.SetFlags(astiav.NewCodecContextFlags(astiav.CodecContextFlagGlobalHeader))
// 	return encoderContext.Open(codec, settings)
// }
