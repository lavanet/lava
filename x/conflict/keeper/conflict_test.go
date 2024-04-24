package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/x/conflict/keeper"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/stretchr/testify/require"
)

func TestValidateSameProviderConflict(t *testing.T) {
	initTester := func() (tester *tester, keeper *keeper.Keeper, ctx sdk.Context) {
		tester = newTester(t)
		keeper = &tester.Keepers.Conflict
		ctx = tester.Ctx
		return
	}

	t.Run("nil check", func(t *testing.T) {
		_, keeper, ctx := initTester()
		_, _, _, err := keeper.ValidateSameProviderConflict(ctx, nil, sdk.AccAddress{})
		require.Error(t, err)

		finalizationConflict := &types.FinalizationConflict{}
		_, _, _, err = keeper.ValidateSameProviderConflict(ctx, finalizationConflict, sdk.AccAddress{})
		require.Error(t, err)
	})

	t.Run("empty reply", func(t *testing.T) {
		_, keeper, ctx := initTester()
		finalizationConflict := &types.FinalizationConflict{
			RelayFinalization_0: &types.RelayFinalization{},
			RelayFinalization_1: &types.RelayFinalization{},
		}
		_, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, sdk.AccAddress{})
		require.Error(t, err)
	})

	t.Run("different provider addr", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(2) // stake provider
		provider0Acc, _ := ts.GetAccount(common.PROVIDER, 0)
		provider1Acc, _ := ts.GetAccount(common.PROVIDER, 1)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, provider0Acc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, provider1Acc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		_, _, _, err = keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
	})

	t.Run("different consumer addr", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)

		consumer, consumerAddr := ts.AddAccount("consumer", 1, 100000)
		_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, ts.plan.Index, 1, false, false)
		require.Nil(ts.T, err)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("different spec", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)

		altSpecName := ts.spec.Index + "PremiumPlus"
		altSpec := ts.AddSpec(altSpecName, common.CreateMockSpec()).Spec(altSpecName)
		altSpec.Index = altSpecName
		altSpec.Name = altSpecName

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("provider not staked", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(0) // stake provider
		providerAcc, _ := ts.AddAccount(common.PROVIDER, 1, 100000)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("empty finalized block hashes", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("non consecutive block hashes", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{
			1: "hash1",
			2: "hash2",
			4: "hash3",
			5: "hash4",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("finalization distance is not right", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
		ts.spec.BlockDistanceForFinalizedData = 1
		ts.AddSpec("modSpec", ts.spec)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{
			1: "hash1",
			2: "hash2",
			3: "hash3",
			4: "hash4",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("all overlapping blocks are equal", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
		ts.spec.BlockDistanceForFinalizedData = 2
		ts.AddSpec("modSpec", ts.spec)

		latestBlockHeight := int64(6)
		finalizationBlockHashes := map[int64]string{
			1: "hash1",
			2: "hash2",
			3: "hash3",
			4: "hash4",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight, finalizationBlockHashes, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("no overlapping", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
		ts.spec.BlockDistanceForFinalizedData = 2
		ts.AddSpec("modSpec", ts.spec)

		latestBlockHeight0 := int64(6)
		finalizationBlockHashes0 := map[int64]string{
			1: "hash1",
			2: "hash2",
			3: "hash3",
			4: "hash4",
		}

		latestBlockHeight1 := int64(10)
		finalizationBlockHashes1 := map[int64]string{
			5: "hash5",
			6: "hash6",
			7: "hash7",
			8: "hash",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight0, finalizationBlockHashes0, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight1, finalizationBlockHashes1, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("overlapping blocks match", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
		ts.spec.BlockDistanceForFinalizedData = 2
		ts.AddSpec("modSpec", ts.spec)

		latestBlockHeight0 := int64(6)
		finalizationBlockHashes0 := map[int64]string{
			1: "hash1",
			2: "hash2",
			3: "hash3",
			4: "hash4",
		}

		latestBlockHeight1 := int64(9)
		finalizationBlockHashes1 := map[int64]string{
			4: "hash4",
			5: "hash5",
			6: "hash6",
			7: "hash7",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight0, finalizationBlockHashes0, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight1, finalizationBlockHashes1, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.Error(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})

	t.Run("overlapping blocks don't match", func(t *testing.T) {
		ts, keeper, ctx := initTester()
		ts.setupForConflict(1) // stake provider
		providerAcc, _ := ts.GetAccount(common.PROVIDER, 0)
		ts.spec.BlockDistanceForFinalizedData = 2
		ts.AddSpec("modSpec", ts.spec)

		latestBlockHeight0 := int64(6)
		finalizationBlockHashes0 := map[int64]string{
			1: "hash1",
			2: "hash2",
			3: "hash3",
			4: "hash4",
		}

		latestBlockHeight1 := int64(9)
		finalizationBlockHashes1 := map[int64]string{
			4: "hash4FAIL",
			5: "hash5",
			6: "hash6",
			7: "hash7",
		}

		reply0, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight0, finalizationBlockHashes0, &ts.spec)
		require.NoError(t, err)
		reply1, err := common.CreateRelayFinalizationForTest(ts.Ctx, ts.consumer, providerAcc, latestBlockHeight1, finalizationBlockHashes1, &ts.spec)
		require.NoError(t, err)
		finalizationConflict := &types.FinalizationConflict{RelayFinalization_0: reply0, RelayFinalization_1: reply1}

		providerAddr, _, _, err := keeper.ValidateSameProviderConflict(ctx, finalizationConflict, ts.consumer.Addr)
		require.NoError(t, err)
		require.Equal(t, providerAcc.Addr.String(), providerAddr.String())
	})
}
