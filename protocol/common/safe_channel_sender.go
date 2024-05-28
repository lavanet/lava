package common

import (
	"context"
	"sync"
)

type SafeChannelSender[T any] struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	ch        chan<- T
	closed    bool
	lock      sync.Mutex
}

func NewSafeChannelSender[T any](ctx context.Context, ch chan<- T) *SafeChannelSender[T] {
	ctx, cancel := context.WithCancel(ctx)
	return &SafeChannelSender[T]{
		ctx:       ctx,
		cancelCtx: cancel,
		ch:        ch,
		closed:    false,
		lock:      sync.Mutex{},
	}
}

func (scs *SafeChannelSender[T]) Send(msg T) {
	scs.lock.Lock()
	defer scs.lock.Unlock()

	select {
	case <-scs.ctx.Done():
		return
	case scs.ch <- msg:
		return
	}
}

func (scs *SafeChannelSender[T]) Close() {
	scs.lock.Lock()
	defer scs.lock.Unlock()

	if scs.closed {
		return
	}

	scs.cancelCtx()
	close(scs.ch)
	scs.closed = true
}
