package transcode

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/simple_webrtc_comm/transcode/internal"
	"github.com/harshabose/tools/buffer/pkg"
)

type MultiConfig struct {
	Steps uint8
	UpdateConfig
}

func (c MultiConfig) validate() error {
	if c.Steps == 0 {
		return fmt.Errorf("steps need be more than 0")
	}
	return c.UpdateConfig.validate()
}

func (c MultiConfig) getBitrates() []int64 {
	bitrates := make([]int64, c.Steps)

	if c.Steps == 1 {
		bitrates[0] = c.MaxBitrate
	} else {
		step := float64(c.MaxBitrate-c.MinBitrate) / float64(c.Steps-1)
		for i := uint8(0); i < c.Steps; i++ {
			bitrates[i] = c.MinBitrate + int64(float64(i)*step)
		}
	}

	return bitrates
}

func NewMultiConfig(minBitrate, maxBitrate int64, steps uint8) MultiConfig {
	c := MultiConfig{
		UpdateConfig: UpdateConfig{
			MaxBitrate: maxBitrate,
			MinBitrate: minBitrate,
		},
		Steps: steps,
	}

	return c
}

type dummyMediaFrameProducer struct {
	buffer buffer.BufferWithGenerator[astiav.Frame]
	CanDescribeMediaFrame
}

func newDummyMediaFrameProducer(buffer buffer.BufferWithGenerator[astiav.Frame], describer CanDescribeMediaFrame) *dummyMediaFrameProducer {
	return &dummyMediaFrameProducer{
		buffer:                buffer,
		CanDescribeMediaFrame: describer,
	}
}

func (p *dummyMediaFrameProducer) pushFrame(ctx context.Context, frame *astiav.Frame) error {
	return p.buffer.Push(ctx, frame)
}

func (p *dummyMediaFrameProducer) GetFrame(ctx context.Context) (*astiav.Frame, error) {
	return p.buffer.Pop(ctx)
}

func (p *dummyMediaFrameProducer) Generate() *astiav.Frame {
	return p.buffer.Generate()
}

func (p *dummyMediaFrameProducer) PutBack(frame *astiav.Frame) {
	p.buffer.PutBack(frame)
}

type splitEncoder struct {
	encoder  *GeneralEncoder
	producer *dummyMediaFrameProducer
}

func newSplitEncoder(encoder *GeneralEncoder, producer *dummyMediaFrameProducer) *splitEncoder {
	return &splitEncoder{
		encoder:  encoder,
		producer: producer,
	}
}

type MultiUpdateEncoder struct {
	encoders []*splitEncoder
	active   atomic.Pointer[splitEncoder]
	config   MultiConfig
	bitrates []int64
	producer CanProduceMediaFrame
	ctx      context.Context
	cancel   context.CancelFunc

	paused   atomic.Bool
	resume   chan struct{}
	pauseMux sync.Mutex
}

func NewMultiUpdateEncoder(ctx context.Context, config MultiConfig, builder *GeneralEncoderBuilder) (*MultiUpdateEncoder, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	ctx2, cancel := context.WithCancel(ctx)
	encoder := &MultiUpdateEncoder{
		encoders: make([]*splitEncoder, 0),
		config:   config,
		bitrates: config.getBitrates(),
		producer: builder.producer,
		ctx:      ctx2,
		cancel:   cancel,
		resume:   make(chan struct{}),
	}

	describer, ok := encoder.producer.(CanDescribeMediaFrame)
	if !ok {
		return nil, ErrorInterfaceMismatch
	}

	initialBitrate, err := builder.GetCurrentBitrate()
	if err != nil {
		initialBitrate = encoder.bitrates[0]
	}

	for _, bitrate := range encoder.bitrates {
		producer := newDummyMediaFrameProducer(buffer.CreateChannelBuffer(ctx2, 90, internal.CreateFramePool()), describer)

		if err := builder.UpdateBitrate(bitrate); err != nil {
			return nil, err
		}

		e, err := builder.BuildWithProducer(ctx2, producer)
		if err != nil {
			return nil, err
		}

		encoder.encoders = append(encoder.encoders, newSplitEncoder(e.(*GeneralEncoder), producer))
	}

	encoder.switchEncoder(encoder.findBestEncoderIndex(initialBitrate))

	return encoder, nil
}

func (u *MultiUpdateEncoder) Ctx() context.Context {
	return u.ctx
}

func (u *MultiUpdateEncoder) Start() {
	for _, encoder := range u.encoders {
		encoder.encoder.Start()
	}

	go u.loop()
}

func (u *MultiUpdateEncoder) GetPacket(ctx context.Context) (*astiav.Packet, error) {
	return u.active.Load().encoder.GetPacket(ctx)
}

func (u *MultiUpdateEncoder) PutBack(packet *astiav.Packet) {
	u.active.Load().encoder.PutBack(packet)
}

func (u *MultiUpdateEncoder) Stop() {
	u.cancel()
}

func (u *MultiUpdateEncoder) UpdateBitrate(bps int64) error {
	if err := u.checkPause(bps); err != nil {
		return err
	}

	bps = u.cutoff(bps)

	bestIndex := u.findBestEncoderIndex(bps)
	u.switchEncoder(bestIndex)

	return nil
}

func (u *MultiUpdateEncoder) findBestEncoderIndex(targetBps int64) int {
	bestIndex := 0
	for i, bitrate := range u.bitrates {
		if bitrate <= targetBps {
			bestIndex = i
		} else {
			break
		}
	}

	return bestIndex
}

func (u *MultiUpdateEncoder) switchEncoder(index int) {
	if index < len(u.encoders) {
		fmt.Printf("swapping to %d encoder with bitrate %d\n", index, u.bitrates[index])
		u.active.Swap(u.encoders[index])
	}
}

func (u *MultiUpdateEncoder) cutoff(bps int64) int64 {
	if bps > u.config.MaxBitrate {
		bps = u.config.MaxBitrate
	}

	if bps < u.config.MinBitrate {
		bps = u.config.MinBitrate
	}

	return bps
}

func (u *MultiUpdateEncoder) shouldPause(bps int64) bool {
	return bps <= u.config.MinBitrate && u.config.CutVideoBelowMinBitrate
}

func (u *MultiUpdateEncoder) checkPause(bps int64) error {
	shouldPause := u.shouldPause(bps)

	if shouldPause {
		fmt.Println("pausing video...")
		return u.PauseEncoding()
	}
	return u.UnPauseEncoding()
}

func (u *MultiUpdateEncoder) PauseEncoding() error {
	u.paused.Store(true)
	return nil
}

func (u *MultiUpdateEncoder) UnPauseEncoding() error {
	u.pauseMux.Lock()
	defer u.pauseMux.Unlock()

	if u.paused.Swap(false) {
		close(u.resume)
		u.resume = make(chan struct{})
	}
	return nil
}

func (u *MultiUpdateEncoder) GetParameterSets() (sps []byte, pps []byte, err error) {
	return u.active.Load().encoder.GetParameterSets()
}

func (u *MultiUpdateEncoder) loop() {
	defer u.close()

	for {
		select {
		case <-u.ctx.Done():
			return
		default:
			frame, err := u.getFrame()
			if err != nil {
				continue
			}

			for _, encoder := range u.encoders {
				if err := u.pushFrame(encoder, frame); err != nil {
					continue
				}
			}

			u.producer.PutBack(frame)
		}
	}
}

func (u *MultiUpdateEncoder) getFrame() (*astiav.Frame, error) {
	ctx, cancel := context.WithTimeout(u.ctx, 50*time.Millisecond)
	defer cancel()

	return u.producer.GetFrame(ctx)
}

func (u *MultiUpdateEncoder) pushFrame(encoder *splitEncoder, frame *astiav.Frame) error {
	if frame == nil {
		return fmt.Errorf("frame is nil from the producer")
	}

	ctx, cancel := context.WithTimeout(u.ctx, 50*time.Millisecond)
	defer cancel()

	refFrame := encoder.producer.Generate()
	if refFrame == nil {
		return fmt.Errorf("failed to generate frame from encoder pool")
	}

	if err := refFrame.Ref(frame); err != nil {
		return fmt.Errorf("erorr while adding ref to frame; err: %s", "refFrame is nil")
	}

	// PUT IN BUFFER
	return encoder.producer.pushFrame(ctx, refFrame)
}

func (u *MultiUpdateEncoder) close() {

}
