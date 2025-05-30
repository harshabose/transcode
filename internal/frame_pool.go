package internal

import (
	"sync"

	"github.com/asticode/go-astiav"

	"github.com/harshabose/tools/buffer/pkg"
)

type framePool struct {
	pool sync.Pool
}

func CreateFramePool() buffer.Pool[astiav.Frame] {
	return &framePool{
		pool: sync.Pool{
			New: func() any {
				return astiav.AllocFrame()
			},
		},
	}
}

func (pool *framePool) Get() *astiav.Frame {
	frame, ok := pool.pool.Get().(*astiav.Frame)

	if frame == nil || !ok {
		return astiav.AllocFrame()
	}
	return frame
}

func (pool *framePool) Put(frame *astiav.Frame) {
	if frame == nil {
		return
	}

	frame.Unref()
	pool.pool.Put(frame)
}

func (pool *framePool) Release() {
	for {
		frame, ok := pool.pool.Get().(*astiav.Frame)
		if frame == nil {
			break
		}
		if !ok {
			continue
		}

		frame.Free()
	}
}
