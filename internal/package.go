package internal

import (
	"context"
	"sync"

	"github.com/asticode/go-astiav"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4/pkg/media"
)

type Pool[T any] interface {
	Get() *T
	Put(*T)
	Release()
}

type Buffer[T any] interface {
	Push(context.Context, *T) error
	Pop(ctx context.Context) (*T, error)
	Size() int
}

type BufferWithGenerator[T any] interface {
	Push(context.Context, *T) error
	Pop(ctx context.Context) (*T, error)
	Size() int
	Generate() *T
	PutBack(*T)
	GetChannel() chan *T
}

func CreateFramePool() Pool[astiav.Frame] {
	return &framePool{
		pool: sync.Pool{
			New: func() any {
				return astiav.AllocFrame()
			},
		},
	}
}

func CreateSamplePool() Pool[media.Sample] {
	return &samplePool{
		pool: sync.Pool{
			New: func() any {
				return &media.Sample{}
			},
		},
	}
}

func CreateRTPPool() Pool[rtp.Packet] {
	return &rtpPool{
		pool: sync.Pool{
			New: func() any {
				return &rtp.Packet{}
			},
		},
	}
}

func CreatePacketPool() Pool[astiav.Packet] {
	return &packetPool{
		pool: sync.Pool{
			New: func() any {
				return astiav.AllocPacket()
			},
		},
	}
}

func CreateChannelBuffer[T any](ctx context.Context, size int, pool Pool[T]) BufferWithGenerator[T] {
	buffer := &limitBuffer[T]{
		pool:          pool,
		bufferChannel: make(chan *T, size),
		inputBuffer:   make(chan *T),
		ctx:           ctx,
	}
	go buffer.loop()
	return buffer
}
