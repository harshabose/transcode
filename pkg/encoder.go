package transcode

//
// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"math"
// 	"time"
//
// 	"github.com/asticode/go-astiav"
//
// 	"github.com/harshabose/tools/buffer/pkg"
//
// 	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
// )
//
// type Encoder struct {
// 	buffer          buffer.BufferWithGenerator[astiav.Packet]
// 	filter          *Filter
// 	ctx             context.Context
// 	codec           *astiav.Codec
// 	encoderContext  *astiav.CodecContext
// 	codecFlags      *astiav.Dictionary
// 	encoderSettings codecSettings
// 	bandwidthChan   chan int64
// 	previousBitrate int64
// 	timer           *time.Timer
// 	testMode        bool // Add this flag
// 	sps             []byte
// 	pps             []byte
// }
//
// func CreateEncoder(ctx context.Context, codecID astiav.CodecID, filter *Filter, options ...EncoderOption) (*Encoder, error) {
// 	encoder := &Encoder{
// 		filter:     filter,
// 		codecFlags: astiav.NewDictionary(),
// 		ctx:        ctx,
// 	}
//
// 	encoder.codec = astiav.FindEncoder(codecID)
// 	if encoder.encoderContext = astiav.AllocCodecContext(encoder.codec); encoder.encoderContext == nil {
// 		return nil, ErrorAllocateCodecContext
// 	}
//
// 	var contextOption EncoderOption
// 	if filter.sinkContext.MediaType() == astiav.MediaTypeAudio {
// 		contextOption = withAudioSetEncoderParameters(filter)
// 	}
// 	if filter.sinkContext.MediaType() == astiav.MediaTypeVideo {
// 		contextOption = withVideoSetEncoderParameters(filter)
// 	}
//
// 	options = append([]EncoderOption{contextOption}, options...)
//
// 	for _, option := range options {
// 		if err := option(encoder); err != nil {
// 			return nil, err
// 		}
// 	}
//
// 	if encoder.encoderSettings == nil {
// 		fmt.Println("warn: no encoder settings are provided")
// 	}
//
// 	encoder.encoderContext.SetFlags(astiav.NewCodecContextFlags(astiav.CodecContextFlagGlobalHeader))
//
// 	if err := encoder.encoderContext.Open(encoder.codec, encoder.codecFlags); err != nil {
// 		return nil, err
// 	}
//
// 	if encoder.buffer == nil {
// 		encoder.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreatePacketPool())
// 	}
//
// 	encoder.findParameterSets(encoder.encoderContext.ExtraData())
//
// 	return encoder, nil
// }
//
// func (encoder *Encoder) Start() {
// 	encoder.timer = time.NewTimer(10 * time.Second)
// 	go encoder.loop()
// }
//
// func (encoder *Encoder) GetParameterSets() ([]byte, []byte) {
// 	return encoder.sps, encoder.pps
// }
//
// func (encoder *Encoder) GetDuration() time.Duration {
// 	if encoder.encoderContext.MediaType() == astiav.MediaTypeAudio {
// 		return time.Duration(float64(time.Second) * float64(encoder.encoderContext.FrameSize()) / float64(encoder.encoderContext.SampleRate()))
// 	}
// 	return time.Duration(float64(time.Second) / encoder.encoderContext.Framerate().Float64())
// }
//
// func (encoder *Encoder) GetTimeBase() astiav.Rational {
// 	return encoder.encoderContext.TimeBase()
// }
//
// func (encoder *Encoder) loop() {
// 	var (
// 		frame  *astiav.Frame
// 		packet *astiav.Packet
// 		err    error
// 	)
// 	defer encoder.close()
//
// loop1:
// 	for {
// 		select {
// 		case <-encoder.ctx.Done():
// 			return
// 		case bitrate := <-encoder.bandwidthChan: // TODO: MIGHT NEED A MUTEX FOR THIS ONE CASE
// 			encoder.UpdateBitrate(bitrate)
// 		case frame = <-encoder.filter.WaitForFrame():
// 			if err = encoder.encoderContext.SendFrame(frame); err != nil {
// 				encoder.filter.PutBack(frame)
// 				if !errors.Is(err, astiav.ErrEagain) {
// 					continue loop1
// 				}
// 			}
// 		loop2:
// 			for {
// 				packet = encoder.buffer.Generate()
// 				if err = encoder.encoderContext.ReceivePacket(packet); err != nil {
// 					encoder.buffer.PutBack(packet)
// 					break loop2
// 				}
//
// 				if err = encoder.pushPacket(packet); err != nil {
// 					encoder.buffer.PutBack(packet)
// 					continue loop2
// 				}
// 			}
// 			encoder.filter.PutBack(frame)
// 		}
// 	}
// }
//
// func (encoder *Encoder) WaitForPacket() chan *astiav.Packet {
// 	return encoder.buffer.GetChannel()
// }
//
// func (encoder *Encoder) pushPacket(packet *astiav.Packet) error {
// 	ctx, cancel := context.WithTimeout(encoder.ctx, time.Second)
// 	defer cancel()
//
// 	return encoder.buffer.Push(ctx, packet)
// }
//
// func (encoder *Encoder) GetPacket() (*astiav.Packet, error) {
// 	ctx, cancel := context.WithTimeout(encoder.ctx, time.Second)
// 	defer cancel()
//
// 	return encoder.buffer.Pop(ctx)
// }
//
// func (encoder *Encoder) PutBack(packet *astiav.Packet) {
// 	encoder.buffer.PutBack(packet)
// }
//
// func (encoder *Encoder) SetBitrateChannel(channel chan int64) {
// 	encoder.bandwidthChan = channel
// }
//
// func (encoder *Encoder) close() {
// 	if encoder.encoderContext != nil {
// 		encoder.encoderContext.Free()
// 	}
// }
//
// func (encoder *Encoder) findParameterSets(extraData []byte) {
// 	if len(extraData) > 0 {
// 		// Find first start code (0x00000001)
// 		for i := 0; i < len(extraData)-4; i++ {
// 			if extraData[i] == 0 && extraData[i+1] == 0 && extraData[i+2] == 0 && extraData[i+3] == 1 {
// 				// Skip start code to get NAL type
// 				nalType := extraData[i+4] & 0x1F
//
// 				// Find next start code or end
// 				nextStart := len(extraData)
// 				for j := i + 4; j < len(extraData)-4; j++ {
// 					if extraData[j] == 0 && extraData[j+1] == 0 && extraData[j+2] == 0 && extraData[j+3] == 1 {
// 						nextStart = j
// 						break
// 					}
// 				}
//
// 				if nalType == 7 { // SPS
// 					encoder.sps = make([]byte, nextStart-i)
// 					copy(encoder.sps, extraData[i:nextStart])
// 				} else if nalType == 8 { // PPS
// 					encoder.pps = make([]byte, len(extraData)-i)
// 					copy(encoder.pps, extraData[i:])
// 				}
//
// 				i = nextStart - 1
// 			}
// 		}
// 		fmt.Println("SPS for current encoder: ", encoder.sps)
// 		fmt.Println("PPS for current encoder: ", encoder.pps)
// 	}
// }
//
// func (encoder *Encoder) UpdateBitrate(bitrate int64) {
// 	// Show current encoder state
// 	currentEncoderBitrate := encoder.encoderContext.BitRate()
// 	fmt.Printf("recommended bitrate update to: %d (previous: %d, encoder actual: %d)\n",
// 		bitrate, encoder.previousBitrate, currentEncoderBitrate)
//
// 	if encoder.previousBitrate == 0 {
// 		encoder.SetBitrate(bitrate)
// 		newEncoderBitrate := encoder.encoderContext.BitRate()
// 		encoder.previousBitrate = bitrate
// 		fmt.Printf("initial bitrate set to: %d (encoder confirms: %d)\n", bitrate, newEncoderBitrate)
// 		return
// 	}
//
// 	change := math.Abs(float64(encoder.previousBitrate - bitrate))
// 	changePercent := change / float64(encoder.previousBitrate) * 100
//
// 	fmt.Printf("bitrate change: %.1f%% (%.0f -> %.0f)\n",
// 		changePercent, float64(encoder.previousBitrate), float64(bitrate))
//
// 	shouldUpdate := false
//
// 	// Much more lenient thresholds for BWE:
// 	if changePercent >= 2.0 { // Was 5.0 - now accepts smaller changes
// 		if changePercent <= 300.0 { // Was 90.0 - now allows recovery
// 			shouldUpdate = true
// 		} else if encoder.previousBitrate <= 200 && bitrate > encoder.previousBitrate {
// 			// Special recovery case: allow any increase from very low bitrates
// 			shouldUpdate = true
// 			fmt.Printf("🔄 recovery mode: very low bitrate, allowing large increase\n")
// 		}
// 	}
//
// 	if shouldUpdate {
// 		oldEncoderBitrate := encoder.encoderContext.BitRate()
// 		encoder.SetBitrate(bitrate)
// 		newEncoderBitrate := encoder.encoderContext.BitRate()
// 		encoder.previousBitrate = bitrate
// 		fmt.Printf("✓ updated encoder bitrate: %d → %d → %d (target → old → new)\n",
// 			bitrate, oldEncoderBitrate, newEncoderBitrate)
// 	} else {
// 		fmt.Printf("✗ bitrate change ignored (%.1f%% change), encoder remains at: %d\n",
// 			changePercent, currentEncoderBitrate)
// 	}
// }
//
// func (encoder *Encoder) SetBitrate(bitrate int64) {
// 	encoder.encoderContext.SetBitRate(bitrate)
// 	// if err := encoder.codecFlags.Set("bitrate", strconv.Itoa(int(bitrate/1000)), 0); err != nil {
// 	// 	fmt.Println("error while setting bitrate; err:", err.Error())
// 	// }
// }
//
// // func (encoder *Encoder) UpdateBitrate(b int64) {
// // 	// Check if timer fired (only happens once)
// // 	select {
// // 	case <-encoder.timer.C:
// // 		fmt.Println("timer hit!!!!!!!!!!!")
// // 		encoder.testMode = true // Set flag
// // 		encoder.SetBitrate(250000000)
// // 		return
// // 	default:
// // 		// Continue to check flag
// // 	}
// //
// // 	// After timer fires, always use 10000
// // 	if encoder.testMode {
// // 		fmt.Println("test mode active - forcing 10000")
// // 		encoder.SetBitrate(250000000)
// // 	}
// //
// // 	// Before timer fires, show current bitrate
// // 	fmt.Println("current bitrate:", encoder.encoderContext.BitRate())
// // }
