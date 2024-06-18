package common

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
)

const retryAttemptsForChannelWrite = 10

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

	shouldBreak := false
	for retry := 0; retry < retryAttemptsForChannelWrite; retry++ {
		select {
		case <-scs.ctx.Done():
		// trying to write to the channel, if the channel is not ready this will fail and retry again up to retryAttemptsForChannelWrite times
		case scs.ch <- msg:
			shouldBreak = true
		default:
			utils.LavaFormatTrace("Failed to send message to channel", utils.LogAttr("attempt", retry))
		}
		if shouldBreak {
			break
		}
		time.Sleep(time.Millisecond) // wait 1 millisecond between each attempt to write to the channel
	}
}

func (scs *SafeChannelSender[T]) ReplaceChannel(ch chan<- T) {
	scs.lock.Lock()
	defer scs.lock.Unlock()

	if scs.closed {
		return
	}

	// check wether the incoming channel is different than the one we currently have.
	// this helps us avoids closing our channel and holding a closed channel causing Close to panic.
	if scs.ch != ch {
		close(scs.ch)
		scs.ch = ch
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
