package common

import (
	"context"
	"fmt"
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
		go func() {
			defer func() { chEnd <- true }()
			select {
			case received, ok := <-ch:
				fmt.Println("got message from channel", ok, received)
				require.True(t, ok)
				require.Equal(t, msg, received)
				return
			case <-time.After(time.Second * 20):
				require.Fail(t, "Expected message to be sent, but channel is empty")
			}
		}()

		// wait for the routine to listen to the channel
		<-time.After(time.Second * 1)
		sender.Send(msg)
		sender.Close()
		require.True(t, sender.closed)
		// wait for the test to end
		<-chEnd
	})
}
