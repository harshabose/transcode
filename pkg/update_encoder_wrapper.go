package transcode

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/asticode/go-astiav"
)

type UpdateConfig struct {
	MaxBitrate, MinBitrate  int64
	CutVideoBelowMinBitrate bool
}

type UpdateEncoder struct {
	encoder Encoder
	config  UpdateConfig
	builder *GeneralEncoderBuilder
	mux     sync.RWMutex
	ctx     context.Context

	paused   atomic.Bool
	resume   chan struct{}
	pauseMux sync.Mutex
}

func NewUpdateEncoder(ctx context.Context, config UpdateConfig, builder *GeneralEncoderBuilder) (*UpdateEncoder, error) {
	updater := &UpdateEncoder{
		config:  config,
		builder: builder,
		resume:  make(chan struct{}),
		ctx:     ctx,
	}

	encoder, err := builder.Build(ctx)
	if err != nil {
		return nil, err
	}

	updater.encoder = encoder

	return updater, nil
}

func (u *UpdateEncoder) Ctx() context.Context {
	u.mux.Lock()
	defer u.mux.Unlock()

	return u.encoder.Ctx()
}

func (u *UpdateEncoder) Start() {
	u.mux.Lock()
	defer u.mux.Unlock()

	u.encoder.Start()
}

func (u *UpdateEncoder) WaitForPacket() chan *astiav.Packet {
	if u.paused.Load() {
		<-u.resume
	}

	return u.encoder.WaitForPacket()
}

func (u *UpdateEncoder) PutBack(packet *astiav.Packet) {
	u.mux.RLock()
	defer u.mux.RUnlock()

	u.encoder.PutBack(packet)
}

func (u *UpdateEncoder) Stop() {
	u.mux.Lock()
	defer u.mux.Unlock()

	u.encoder.Stop()
}

// UpdateBitrate modifies the encoder's target bitrate to the specified value in bits per second.
// Returns an error if the update fails.
func (u *UpdateEncoder) UpdateBitrate(bps int64) error {
	if err := u.checkPause(bps); err != nil {
		return err
	}

	bps = u.cutoff(bps)

	g, ok := u.encoder.(CanGetCurrentBitrate)
	if !ok {
		return ErrorInterfaceMismatch
	}

	current, err := g.GetCurrentBitrate()
	if err != nil {
		return err
	}
	fmt.Printf("got bitrate update request (%d -> %d)\n", current, bps)

	_, change := u.calculateBitrateChange(current, bps)
	if change < 5 {
		return nil
	}

	if err := u.builder.UpdateBitrate(bps); err != nil {
		return err
	}

	newEncoder, err := u.builder.Build(u.ctx)
	if err != nil {
		return fmt.Errorf("build new encoder: %w", err)
	}

	newEncoder.Start()

	// Wait for the first packet from the new encoder
	// firstPacket := <-newEncoder.WaitForPacket()

	u.mux.Lock()
	oldEncoder := u.encoder
	u.encoder = newEncoder
	u.mux.Unlock()

	// Put the first packet back for next WaitForPacket()
	// newEncoder.PutBack(firstPacket)

	if oldEncoder != nil {
		oldEncoder.Stop()
	}

	// Print encoder update notification
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════╗")
	fmt.Println("║        🎥 ENCODER UPDATED 🎥          ║")
	fmt.Printf("║      New Bitrate: %6d kbps        ║\n", bps/1000)
	fmt.Println("╚═══════════════════════════════════════╝")
	fmt.Println()

	return nil
}

func (u *UpdateEncoder) cutoff(bps int64) int64 {
	if bps > u.config.MaxBitrate {
		bps = u.config.MaxBitrate
	}

	if bps < u.config.MinBitrate {
		bps = u.config.MinBitrate
	}

	return bps
}

func (u *UpdateEncoder) shouldPause(bps int64) bool {
	return bps <= u.config.MinBitrate && u.config.CutVideoBelowMinBitrate
}

func (u *UpdateEncoder) checkPause(bps int64) error {
	shouldPause := u.shouldPause(bps)

	if shouldPause {
		fmt.Println("pausing video...")
		return u.PauseEncoding()
	}
	return u.UnPauseEncoding()
}

func (u *UpdateEncoder) PauseEncoding() error {
	u.paused.Store(true)
	return nil
}

func (u *UpdateEncoder) UnPauseEncoding() error {
	u.pauseMux.Lock()
	defer u.pauseMux.Unlock()

	if u.paused.Swap(false) {
		close(u.resume)
		u.resume = make(chan struct{})
	}
	return nil
}

func (u *UpdateEncoder) GetParameterSets() (sps []byte, pps []byte, err error) {
	p, ok := u.encoder.(CanGetParameterSets)
	if !ok {
		return nil, nil, ErrorInterfaceMismatch
	}

	return p.GetParameterSets()
}

func (u *UpdateEncoder) calculateBitrateChange(currentBps, newBps int64) (absoluteChange int64, percentageChange float64) {
	absoluteChange = newBps - currentBps
	if absoluteChange < 0 {
		absoluteChange = -absoluteChange
	}

	if currentBps > 0 {
		percentageChange = (float64(absoluteChange) / float64(currentBps)) * 100
	}

	return absoluteChange, percentageChange
}

func (u *UpdateEncoder) swapSoon() {

}
