package internal

import (
	"sync"

	"github.com/pion/rtp"
)

type rtpPool struct {
	pool sync.Pool
}

func (pool *rtpPool) Get() *rtp.Packet {
	packet, ok := pool.pool.Get().(*rtp.Packet)

	if packet == nil || !ok {
		return &rtp.Packet{}
	}
	return packet
}

func (pool *rtpPool) Put(packet *rtp.Packet) {
	if packet == nil {
		return
	}
	pool.pool.Put(packet)
}

func (pool *rtpPool) Release() {
	for {
		packet, ok := pool.pool.Get().(*rtp.Packet)
		if !ok {
			continue
		}
		if packet == nil {
			break
		}

		packet = nil
	}
}
