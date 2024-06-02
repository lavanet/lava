package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeChannelSender(t *testing.T) {
	t.Run("Send message", func(t *testing.T) {
		ctx := context.Background()
		ch := make(chan int)
		sender := NewSafeChannelSender(ctx, ch)

		msg := 42

		go func() {
			select {
			case received, ok := <-ch:
				require.True(t, ok)
				require.Equal(t, msg, received)
				return
			default:
				require.Fail(t, "Expected message to be sent, but channel is empty")
			}
		}()

		sender.Send(msg)
		sender.Close()
		require.True(t, sender.closed)

		require.Panics(t, func() {
			sender.Send(msg)
		})
	})
}
