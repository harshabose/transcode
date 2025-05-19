package transcode

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
	"github.com/harshabose/tools/buffer/pkg"
)

type Encoder struct {
	buffer         buffer.BufferWithGenerator[astiav.Packet]
	filter         *Filter
	codec          *astiav.Codec
	codecFlags     *astiav.Dictionary
	copyCodecFlags *astiav.Dictionary
	codecSettings  codecSettings
	bandwidthChan  chan int64
	options        []EncoderOption
	sps, pps       []byte

	encoderContext         *astiav.CodecContext
	fallbackEncoderContext *astiav.CodecContext

	ctx context.Context
	mux sync.Mutex
}

func NewEncoder(ctx context.Context, codecID astiav.CodecID, filter *Filter, options ...EncoderOption) (*Encoder, error) {
	encoder := &Encoder{
		filter:     filter,
		codecFlags: astiav.NewDictionary(),
		ctx:        ctx,
	}
	if encoder.codec = astiav.FindEncoder(codecID); encoder.codec == nil {
		return nil, ErrorNoCodecFound
	}

	encoderContext, err := createNewEncoder(encoder.codec, filter)
	if err != nil {
		return nil, err
	}
	encoder.encoderContext = encoderContext

	for _, option := range options {
		if err := option(encoder); err != nil {
			return nil, err
		}
	}

	if encoder.codecSettings == nil {
		fmt.Println("warn: no encoder settings were provided")
	}

	copyDict, err := copyDictionary(encoder.codecFlags)
	if err != nil {
		return nil, err
	}
	encoder.copyCodecFlags = copyDict

	if err := openEncoder(encoder.encoderContext, encoder.codec, encoder.codecFlags); err != nil {
		return nil, err
	}

	if encoder.buffer == nil {
		encoder.buffer = buffer.CreateChannelBuffer(ctx, 256, internal.CreatePacketPool())
	}

	encoder.findParameterSets(encoder.encoderContext.ExtraData())

	return encoder, nil
}

func (e *Encoder) Start() {
	go e.loop()
}

func (e *Encoder) GetPacket() (*astiav.Packet, error) {
	ctx, cancel := context.WithTimeout(e.ctx, time.Second) // TODO: Needs to be based on something
	defer cancel()

	return e.buffer.Pop(ctx)
}

func (e *Encoder) WaitForPacket() chan *astiav.Packet {
	return e.buffer.GetChannel()
}

func (e *Encoder) pushPacket(packet *astiav.Packet) error {
	ctx, cancel := context.WithTimeout(e.ctx, time.Second)
	defer cancel()

	return e.buffer.Push(ctx, packet)
}

func (e *Encoder) PutBack(packet *astiav.Packet) {
	e.buffer.PutBack(packet)
}

func (e *Encoder) GetParameterSets() (sps []byte, pps []byte) {
	sps = e.sps
	pps = e.pps

	return sps, pps
}

func (e *Encoder) GetTimeBase() astiav.Rational {
	e.mux.Lock()
	defer e.mux.Unlock()

	if e.encoderContext != nil {
		return e.encoderContext.TimeBase()
	}
	if e.fallbackEncoderContext != nil {
		return e.fallbackEncoderContext.TimeBase()
	}

	return astiav.Rational{}
}

func (e *Encoder) GetDuration() time.Duration {
	e.mux.Lock()
	defer e.mux.Unlock()

	if e.encoderContext != nil {
		if e.encoderContext.MediaType() == astiav.MediaTypeAudio {
			return time.Duration(float64(time.Second) * float64(e.encoderContext.FrameSize()) / float64(e.encoderContext.SampleRate()))
		}
		return time.Duration(float64(time.Second) / e.encoderContext.Framerate().Float64())

	}

	if e.fallbackEncoderContext != nil {
		if e.fallbackEncoderContext.MediaType() == astiav.MediaTypeAudio {
			return time.Duration(float64(time.Second) * float64(e.fallbackEncoderContext.FrameSize()) / float64(e.fallbackEncoderContext.SampleRate()))
		}
		return time.Duration(float64(time.Second) / e.fallbackEncoderContext.Framerate().Float64())
	}

	return time.Second / 30
}

func (e *Encoder) SetBitrateChannel(channel chan int64) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.bandwidthChan = channel
}

func (e *Encoder) createNewEncoderContext() error {
	e.mux.Lock()

	e.fallbackEncoderContext = e.encoderContext
	e.encoderContext = nil
	copyDict, err := copyDictionary(e.copyCodecFlags)
	if err != nil {
		e.mux.Unlock()
		return err
	}

	e.codecFlags.Free()
	e.codecFlags = nil
	e.codecFlags = copyDict

	e.mux.Unlock()

	encoderContext, err := createNewOpenEncoder(e.codec, e.filter, e.codecFlags)
	if err != nil {
		e.mux.Lock()
		e.encoderContext = e.fallbackEncoderContext
		e.fallbackEncoderContext = nil
		e.mux.Unlock()

		fmt.Printf("New encoder creation failed, reverted: %v\n", err)
		return err
	}

	e.mux.Lock()
	oldFallback := e.fallbackEncoderContext
	e.encoderContext = encoderContext
	e.fallbackEncoderContext = nil // Free later
	e.mux.Unlock()

	if oldFallback != nil {
		oldFallback.Free()
		oldFallback = nil
		fmt.Printf("🧹 Cleaned up fallback encoder context\n")
	}

	return nil
}

func (e *Encoder) getCurrentBitrate() (int64, error) {
	// Get the x264opts string
	entry := e.copyCodecFlags.Get("x264opts", nil, 0)
	if entry == nil {
		return 0, errors.New("error getting x264opts from the dictionary") // Default value
	}

	x264opts := entry.Value()

	// Parse bitrate from "bitrate=2500:vbv-maxrate=2500:..."
	parts := strings.Split(x264opts, ":")
	for _, part := range parts {
		if strings.HasPrefix(part, "bitrate=") {
			bitrateStr := strings.TrimPrefix(part, "bitrate=")
			bitrate, err := strconv.ParseInt(bitrateStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return bitrate, nil
		}
	}

	return 2500, errors.New("cannot find bitrate in the dictionary") // Default if not found
}

func (e *Encoder) updateX264OptsWithNewBitrate(newBitrate int64) error {
	entry := e.copyCodecFlags.Get("x264opts", nil, 0)
	if entry == nil {
		return errors.New("x264opts not found")
	}

	x264opts := entry.Value()
	parts := strings.Split(x264opts, ":")

	// Find and replace bitrate part
	for i, part := range parts {
		if strings.HasPrefix(part, "bitrate=") {
			parts[i] = fmt.Sprintf("bitrate=%d", newBitrate)
			break
		}
	}

	newX264opts := strings.Join(parts, ":")
	return e.copyCodecFlags.Set("x264opts", newX264opts, 0)
}

// updateBitrate updates the bitrate on codecFlags. The bitrate units are kbps (kilobits per second)
func (e *Encoder) updateBitrate(bitrate int64) error {
	start := time.Now()

	e.mux.Lock()

	current, err := e.getCurrentBitrate()
	if err != nil {
		e.mux.Unlock()
		fmt.Println("error getting current bitrate; err:", err.Error())
		return err
	}

	change := math.Abs(float64(current)-float64(bitrate)) / math.Abs(float64(current))

	if change < 0.05 {
		e.mux.Unlock()
		fmt.Printf("change not appropriate; current: %d; new: %d; change:%f\n", current, bitrate, change)
		return nil
	}

	fmt.Println("change approved!!!!!!!!!!!!!!!!!!!!!!!!; change:", change)

	// NOTE: ONLY UPDATE IF CHANGE IS MORE THAN 10% AND LESS THAN 200%
	if err := e.updateX264OptsWithNewBitrate(bitrate); err != nil {
		e.mux.Unlock()
		fmt.Println("error while updating the bitrate; err:", err.Error())
		return err
	}

	e.mux.Unlock()
	if err := e.createNewEncoderContext(); err != nil {
		return err
	}

	duration := time.Since(start)
	fmt.Printf("🔄 Bitrate updated: %d → %d (%.1f%%) in %v\n",
		current, bitrate, change*100, duration)

	return nil
}

func (e *Encoder) pickContextAndProcess(frame *astiav.Frame) error {
	e.mux.Lock()
	defer e.mux.Unlock()

	if e.encoderContext != nil {
		if err := e.sendFrameAndPutPackets(e.encoderContext, frame); err != nil {
			return err
		}

		return nil
	}

	if e.fallbackEncoderContext != nil {
		if err := e.sendFrameAndPutPackets(e.fallbackEncoderContext, frame); err != nil {
			return err
		}

		return nil
	}

	return errors.New("invalid encoder context state")
}

func (e *Encoder) sendFrameAndPutPackets(encoderContext *astiav.CodecContext, frame *astiav.Frame) error {
	// NOTE: MUX NOT NEEDED AS BUFFER IS NON-MUX IMPLEMENTATION
	// TODO: DO I NEED MUX?
	// NOTE: IF THE CALLED OF THIS FUNCTION LOCKS, DOES THE LOCK STILL PERSIST HERE?
	defer e.filter.PutBack(frame)

	if err := encoderContext.SendFrame(frame); err != nil {
		return err
	}

	for {
		packet := e.buffer.Generate()
		if err := encoderContext.ReceivePacket(packet); err != nil {
			e.buffer.PutBack(packet)
			break
		}

		if err := e.pushPacket(packet); err != nil {
			e.buffer.PutBack(packet)
			continue
		}
	}

	return nil
}

func (e *Encoder) loop() {
	defer e.Close()

	for {
		select {
		case <-e.ctx.Done():
			return
		case bitrate := <-e.bandwidthChan:
			if err := e.updateBitrate(bitrate); err != nil {
				fmt.Printf("error while encoding; err: %s\n", err.Error())
			}
		case frame := <-e.filter.WaitForFrame():
			if err := e.pickContextAndProcess(frame); err != nil {
				if !errors.Is(err, astiav.ErrEagain) {
					continue
				}
			}
		}
	}
}

func (e *Encoder) Close() {
	e.mux.Lock()
	defer e.mux.Unlock()

	if e.encoderContext != nil {
		e.encoderContext.Free()
		e.encoderContext = nil
	}

	if e.fallbackEncoderContext != nil {
		e.fallbackEncoderContext.Free()
		e.encoderContext = nil
	}
}

func (e *Encoder) findParameterSets(extraData []byte) {
	if len(extraData) > 0 {
		// Find the first start code (0x00000001)
		for i := 0; i < len(extraData)-4; i++ {
			if extraData[i] == 0 && extraData[i+1] == 0 && extraData[i+2] == 0 && extraData[i+3] == 1 {
				// Skip start code to get the NAL type
				nalType := extraData[i+4] & 0x1F

				// Find the next start code or end
				nextStart := len(extraData)
				for j := i + 4; j < len(extraData)-4; j++ {
					if extraData[j] == 0 && extraData[j+1] == 0 && extraData[j+2] == 0 && extraData[j+3] == 1 {
						nextStart = j
						break
					}
				}

				if nalType == 7 { // SPS
					e.sps = make([]byte, nextStart-i)
					copy(e.sps, extraData[i:nextStart])
				} else if nalType == 8 { // PPS
					e.pps = make([]byte, len(extraData)-i)
					copy(e.pps, extraData[i:])
				}

				i = nextStart - 1
			}
		}
		fmt.Println("SPS for current encoder: ", e.sps)
		fmt.Println("PPS for current encoder: ", e.pps)
	}
}

func createNewEncoder(codec *astiav.Codec, filter *Filter) (*astiav.CodecContext, error) {
	encoderContext := astiav.AllocCodecContext(codec)
	if encoderContext == nil {
		return nil, ErrorAllocateCodecContext
	}

	if filter.sinkContext.MediaType() == astiav.MediaTypeAudio {
		withAudioSetEncoderContextParameters(filter, encoderContext)
	}
	if filter.sinkContext.MediaType() == astiav.MediaTypeVideo {
		withVideoSetEncoderContextParameter(filter, encoderContext)
	}

	return encoderContext, nil
}

func createNewOpenEncoder(codec *astiav.Codec, filter *Filter, settings *astiav.Dictionary) (*astiav.CodecContext, error) {
	encoderContext, err := createNewEncoder(codec, filter)
	if err != nil {
		return nil, err
	}

	if err := openEncoder(encoderContext, codec, settings); err != nil {
		return nil, err
	}

	return encoderContext, nil
}

func openEncoder(encoderContext *astiav.CodecContext, codec *astiav.Codec, settings *astiav.Dictionary) error {
	encoderContext.SetFlags(astiav.NewCodecContextFlags(astiav.CodecContextFlagGlobalHeader))
	if err := encoderContext.Open(codec, settings); err != nil {
		return err
	}

	return nil
}

func copyDictionary(source *astiav.Dictionary) (*astiav.Dictionary, error) {
	copyBytes := source.Pack()
	newDict := astiav.NewDictionary()

	if err := newDict.Unpack(copyBytes); err != nil {
		return nil, err
	}

	return newDict, nil
}
