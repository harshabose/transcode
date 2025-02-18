package internal

import (
	"context"
	"fmt"
)

type limitBuffer[T any] struct {
	pool          Pool[T]
	bufferChannel chan *T
	inputBuffer   chan *T
	ctx           context.Context
}

func (buffer *limitBuffer[T]) Push(ctx context.Context, element *T) error {
	select {
	case buffer.inputBuffer <- element:
		// WARN: LACKS CHECKS FOR CLOSED CHANNEL
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (buffer *limitBuffer[T]) Pop(ctx context.Context) (*T, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case data, ok := <-buffer.bufferChannel:
		if !ok {
			return nil, ErrorChannelBufferClose
		}
		if data == nil {
			return nil, ErrorElementUnallocated
		}
		return data, nil
	}
}

func (buffer *limitBuffer[T]) Generate() *T {
	return buffer.pool.Get()
}

func (buffer *limitBuffer[T]) PutBack(element *T) {
	if buffer.pool != nil {
		buffer.pool.Put(element)
	}
}

func (buffer *limitBuffer[T]) GetChannel() chan *T {
	return buffer.bufferChannel
}

func (buffer *limitBuffer[T]) Size() int {
	return len(buffer.bufferChannel)
}

func (buffer *limitBuffer[T]) loop() {
	defer buffer.close()
loop:
	for {
		select {
		case <-buffer.ctx.Done():
			return
		case element, ok := <-buffer.inputBuffer:
			if !ok || element == nil {
				continue loop
			}
			select {
			case buffer.bufferChannel <- element: // SUCCESSFULLY BUFFERED
				continue loop
			default:
				select {
				case oldElement := <-buffer.bufferChannel:
					buffer.PutBack(oldElement)
					select {
					case buffer.bufferChannel <- element:
						continue loop
					default:
						fmt.Println("unexpected buffer state. skipping the element..")
						buffer.PutBack(element)
					}
				}
			}
		}
	}
}

func (buffer *limitBuffer[T]) close() {
loop:
	for {
		select {
		case element := <-buffer.bufferChannel:
			if buffer.pool != nil {
				buffer.pool.Put(element)
			}
		default:
			close(buffer.bufferChannel)
			close(buffer.inputBuffer)
			break loop
		}
	}
	if buffer.pool != nil {
		buffer.pool.Release()
	}
}
