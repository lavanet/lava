package rpcprovider

import (
	"context"
	"sync"
)

type SafeChannelSender[T any] struct {
	ctx  context.Context
	ch   chan<- T
	open bool
	lock sync.Mutex
}

func NewSafeChannelSender[T any](ctx context.Context, ch chan<- T) *SafeChannelSender[T] {
	return &SafeChannelSender[T]{
		ctx:  ctx,
		ch:   ch,
		open: true,
		lock: sync.Mutex{},
	}
}

func (scs *SafeChannelSender[T]) Send(msg T) {
	scs.lock.Lock()
	defer scs.lock.Unlock()

	select {
	case <-scs.ctx.Done():
		return
	default:
		if scs.open {
			go func() { scs.ch <- msg }() // This is inside a goroutine because the select...case is called after this function returns
		}
	}
}

func (scs *SafeChannelSender[T]) Close() {
	scs.lock.Lock()
	defer scs.lock.Unlock()

	close(scs.ch)
	scs.open = true
}
