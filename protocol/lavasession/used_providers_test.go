package lavasession

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/gogo/status"
	"github.com/lavanet/lava/v5/protocol/common"
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
		unwanted := usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
		require.Len(t, unwanted, 0)
		consumerSessionsMap := ConsumerSessionsMap{"test": &SessionInfo{}, "test2": &SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		canUseAgain = usedProviders.tryLockSelection()
		require.True(t, canUseAgain)
		unwanted = usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
		require.Len(t, unwanted, 2)
		require.Equal(t, 2, usedProviders.CurrentlyUsed())
		canUseAgain = usedProviders.tryLockSelection()
		require.False(t, canUseAgain)
		consumerSessionsMap = ConsumerSessionsMap{"test3": &SessionInfo{}, "test4": &SessionInfo{}}
		usedProviders.AddUsed(consumerSessionsMap, nil)
		unwanted = usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
		require.Len(t, unwanted, 4)
		require.Equal(t, 4, usedProviders.CurrentlyUsed())
		// one provider gives a retry
		usedProviders.RemoveUsed("test", NewRouterKey(nil), status.Error(codes.Code(SessionOutOfSyncGRPCCode), ""))
		require.Equal(t, 3, usedProviders.CurrentlyUsed())
		unwanted = usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
		require.Len(t, unwanted, 3)
		// one provider gives a result
		usedProviders.RemoveUsed("test2", NewRouterKey(nil), nil)
		unwanted = usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
		require.Len(t, unwanted, 3)
		require.Equal(t, 2, usedProviders.CurrentlyUsed())
		// one provider gives an error
		usedProviders.RemoveUsed("test3", NewRouterKey(nil), fmt.Errorf("bad"))
		unwanted = usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
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
		unwanted := usedProviders.GetUnwantedProvidersToSend(NewRouterKey(nil))
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
		require.True(t, errors.Is(canUseAgain, ContextDoneNoNeedToLockSelectionError))
	})
}

// TestDecideEligibility verifies the eligibility logic used by RemoveUsed.
func TestDecideEligibility(t *testing.T) {
	t.Run("unsupported method marks unwanted", func(t *testing.T) {
		unsupportedErrors := []error{
			fmt.Errorf("method not found"),
			fmt.Errorf("endpoint not found"),
			fmt.Errorf("method not supported"),
		}

		for _, err := range unsupportedErrors {
			isUnsupported := common.IsUnsupportedMethodError("", 0, err.Error())
			isSyncLoss := IsSessionSyncLoss(err)
			result := common.DecideEligibility(isUnsupported, isSyncLoss, !isSyncLoss)
			require.Equal(t, common.EligibilityMarkUnwanted, result.Action,
				"Should mark unwanted for unsupported method: %s", err.Error())
		}
	})

	t.Run("first sync loss allows retry", func(t *testing.T) {
		err := status.Error(codes.Code(SessionOutOfSyncGRPCCode), "session out of sync")
		isUnsupported := common.IsUnsupportedMethodError("", 0, err.Error())
		isSyncLoss := IsSessionSyncLoss(err)
		result := common.DecideEligibility(isUnsupported, isSyncLoss, true)
		require.Equal(t, common.EligibilityAllowRetry, result.Action,
			"Should allow retry on first sync loss")
	})

	t.Run("second sync loss marks unwanted", func(t *testing.T) {
		err := status.Error(codes.Code(SessionOutOfSyncGRPCCode), "session out of sync")
		isUnsupported := common.IsUnsupportedMethodError("", 0, err.Error())
		isSyncLoss := IsSessionSyncLoss(err)
		result := common.DecideEligibility(isUnsupported, isSyncLoss, false)
		require.Equal(t, common.EligibilityMarkUnwanted, result.Action,
			"Should mark unwanted on second sync loss")
	})

	t.Run("normal errors mark unwanted", func(t *testing.T) {
		normalErrors := []error{
			fmt.Errorf("execution reverted: some error"),
			fmt.Errorf("internal server error"),
			fmt.Errorf("timeout"),
		}

		for _, err := range normalErrors {
			isUnsupported := common.IsUnsupportedMethodError("", 0, err.Error())
			isSyncLoss := IsSessionSyncLoss(err)
			result := common.DecideEligibility(isUnsupported, isSyncLoss, true)
			require.Equal(t, common.EligibilityMarkUnwanted, result.Action,
				"Should mark unwanted for normal error: %s", err.Error())
		}
	})
}
