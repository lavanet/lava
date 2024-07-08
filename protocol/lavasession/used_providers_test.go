package lavasession

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestUsedProviders(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		usedProviders := NewUsedProviders(nil)
		canUse := usedProviders.tryLockSelection()
		require.True(t, canUse)
		canUseAgain := usedProviders.tryLockSelection()
		require.False(t, canUseAgain)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		unwanted := usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 0)
		consumerSessionsMap := ConsumerSessionsMap{"test": &SessionInfo{}, "test2": &SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		canUseAgain = usedProviders.tryLockSelection()
		require.True(t, canUseAgain)
		unwanted = usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 2)
		require.Equal(t, 2, usedProviders.CurrentlyUsed())
		canUseAgain = usedProviders.tryLockSelection()
		require.False(t, canUseAgain)
		consumerSessionsMap = ConsumerSessionsMap{"test3": &SessionInfo{}, "test4": &SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		unwanted = usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 4)
		require.Equal(t, 4, usedProviders.CurrentlyUsed())
		// one provider gives a retry
		usedProviders.RemoveUsed("test", status.Error(codes.Code(SessionOutOfSyncError.ABCICode()), ""))
		require.Equal(t, 3, usedProviders.CurrentlyUsed())
		unwanted = usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 3)
		// one provider gives a result
		usedProviders.RemoveUsed("test2", nil)
		unwanted = usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 3)
		require.Equal(t, 2, usedProviders.CurrentlyUsed())
		// one provider gives an error
		usedProviders.RemoveUsed("test3", fmt.Errorf("bad"))
		unwanted = usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 3)
		require.Equal(t, 1, usedProviders.CurrentlyUsed())
		canUseAgain = usedProviders.tryLockSelection()
		require.True(t, canUseAgain)
	})
}

func TestUsedProvidersAsync(t *testing.T) {
	t.Run("concurrency", func(t *testing.T) {
		usedProviders := NewUsedProviders(nil)
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		go func() {
			time.Sleep(time.Millisecond * 10)
			consumerSessionsMap := ConsumerSessionsMap{"test": &SessionInfo{}, "test2": &SessionInfo{}}
			usedProviders.AddUsed(consumerSessionsMap, nil)
		}()
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		canUseAgain := usedProviders.TryLockSelection(ctx)
		require.Nil(t, canUseAgain)
		unwanted := usedProviders.GetUnwantedProvidersToSend()
		require.Len(t, unwanted, 2)
		require.Equal(t, 2, usedProviders.CurrentlyUsed())
	})
}

func TestUsedProvidersAsyncFail(t *testing.T) {
	t.Run("concurrency", func(t *testing.T) {
		usedProviders := NewUsedProviders(nil)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		canUseAgain := usedProviders.TryLockSelection(ctx)
		require.Error(t, canUseAgain)
	})
}

func TestUsedProviderContextTimeout(t *testing.T) {
	t.Run("concurrency", func(t *testing.T) {
		usedProviders := NewUsedProviders(nil)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		canUse := usedProviders.TryLockSelection(ctx)
		require.Nil(t, canUse)
		require.Zero(t, usedProviders.CurrentlyUsed())
		require.Zero(t, usedProviders.SessionsLatestBatch())
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()
		canUseAgain := usedProviders.TryLockSelection(ctx)
		require.Error(t, canUseAgain)
		require.True(t, ContextDoneNoNeedToLockSelectionError.Is(canUseAgain))
	})
}
