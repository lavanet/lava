package common

import (
	"context"
	"sync"

	"github.com/lavanet/lava/utils"
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

	if scs.closed {
		utils.LavaFormatTrace("Attempted to send message to closed channel")
		return
	}

	select {
	case <-scs.ctx.Done():
	case scs.ch <- msg:
	default:
		utils.LavaFormatTrace("Failed to send message to channel")
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
