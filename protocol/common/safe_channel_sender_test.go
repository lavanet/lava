package common

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSafeChannelSender(t *testing.T) {
	t.Run("Send message", func(t *testing.T) {
		ctx := context.Background()
		ch := make(chan int)
		sender := NewSafeChannelSender(ctx, ch)

		msg := 42
		chEnd := make(chan bool)
		ready := make(chan bool)
		go func() {
			ready <- true // Signal that goroutine is starting
			defer func() { chEnd <- true }()
			select {
			case received, ok := <-ch:
				require.True(t, ok)
				require.Equal(t, msg, received)
				return
			case <-time.After(time.Second * 20):
				require.Fail(t, "Expected message to be sent, but channel is empty")
			}
		}()

		// wait for the routine to be ready
		<-ready
		// Give it a small additional time to ensure it's listening on the channel
		time.Sleep(10 * time.Millisecond)
		sender.Send(msg)
		sender.Close()
		require.True(t, sender.closed)
		// wait for the test to end
		<-chEnd
	})
}
