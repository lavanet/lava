package keeper_test

import (
	"strconv"
	"testing"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/stretchr/testify/require"
)

// TestParseIprpcOverIbcMemo tests the behavior of OnRecvPacket() for different memos:
// 0. empty memo -> "not an iprpc memo" error
// 1. non-JSON memo -> "not an iprpc memo" error
// 2. JSON memo without "iprpc" tag -> "not an iprpc memo" error
// 3. valid JSON memo with "iprpc" tag -> happy flow
// 4. invalid JSON memo with "iprpc" tag (invalid/missing values) -> returns error (multiple cases)
func TestParseIprpcOverIbcMemo(t *testing.T) {
	ts := newTester(t, false)
	memos := []string{
		"",
		"blabla",
		`{
			"client": "Bruce",
		    "duration": 3	
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "",
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "spec": "mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "other-mockspec",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "duration": 3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": -3
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec"
			}
		}`,
	}

	const (
		EMPTY = iota
		NOT_JSON
		JSON_NO_IPRPC
		VALID_JSON_IPRPC
		INVALID_CREATOR_JSON_IPRPC
		MISSING_CREATOR_JSON_IPRPC
		INVALID_SPEC_JSON_IPRPC
		MISSING_SPEC_JSON_IPRPC
		INVALID_DURATION_JSON_IPRPC
		MISSING_DURATION_JSON_IPRPC
	)

	testCases := []struct {
		name         string
		memoInd      int
		expectError  *sdkerrors.Error
		expectedMemo types.IprpcMemo
	}{
		{
			name:         "empty memo",
			memoInd:      EMPTY,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo not json",
			memoInd:      NOT_JSON,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo json that is not iprpc",
			memoInd:      JSON_NO_IPRPC,
			expectError:  types.ErrMemoNotIprpcOverIbc,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "memo iprpc json valid",
			memoInd:      VALID_JSON_IPRPC,
			expectError:  nil,
			expectedMemo: types.IprpcMemo{Creator: "my-moniker", Spec: "mockspec", Duration: 3},
		},
		{
			name:         "invalid memo iprpc json - invalid creator",
			memoInd:      INVALID_CREATOR_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing creator",
			memoInd:      MISSING_CREATOR_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid spec",
			memoInd:      INVALID_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing spec",
			memoInd:      MISSING_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid duration",
			memoInd:      INVALID_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - missing duration",
			memoInd:      MISSING_SPEC_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			data := ts.createIbcTransferPacketData(memos[tt.memoInd])
			memo, err := ts.Keepers.Rewards.ExtractIprpcMemoFromPacket(ts.Ctx, data)
			require.True(t, tt.expectError.Is(err))
			require.True(t, memo.IsEqual(tt.expectedMemo))
		})
	}
}

// Prevent strconv unused error
var _ = strconv.IntSize

func createNPendingIprpcFunds(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PendingIprpcFund {
	items := make([]types.PendingIprpcFund, n)
	for i := range items {
		items[i] = types.PendingIprpcFund{
			Index:       uint64(i),
			Creator:     "dummy",
			Spec:        "mock",
			Month:       1,
			Expiry:      uint64(ctx.BlockTime().UTC().Unix()) + uint64(i),
			CostCovered: sdk.NewCoin("denom", math.OneInt()),
		}
		keeper.SetPendingIprpcFund(ctx, items[i])
	}
	return items
}

func TestPendingIprpcFundsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		require.True(t, found)
	}
}

func TestPendingIprpcFundsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePendingIprpcFund(ctx, item.Index)
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		require.False(t, found)
	}
}

func TestPendingIprpcFundsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPendingIprpcFund(ctx)),
	)
}

func TestPendingIprpcFundsRemoveExpired(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(3 * time.Second))
	keeper.RemoveExpiredPendingIprpcFund(ctx)
	for _, item := range items {
		_, found := keeper.GetPendingIprpcFund(ctx, item.Index)
		if item.Index <= 3 {
			require.False(t, found)
		} else {
			require.True(t, found)
		}
	}
}

func TestPendingIprpcFundGetLatest(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	require.True(t, latest.IsEmpty())
	items := createNPendingIprpcFunds(keeper, ctx, 10)
	latest = keeper.GetLatestPendingIprpcFund(ctx)
	require.True(t, latest.IsEqual(items[len(items)-1]))
}

func TestPendingIprpcFundNew(t *testing.T) {
	ts := newTester(t, false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	spec := ts.Spec("mock")
	err := keeper.NewPendingIprpcFund(ctx, "creator", "eth", 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.Error(t, err) // error for non-existent spec
	err = keeper.NewPendingIprpcFund(ctx, "creator", "eth", 1, nil)
	require.Error(t, err) // error for invalid funds
	err = keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	err = keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	require.Equal(t, uint64(1), latest.Index) // latest is the second object with index 1
}

func TestPendingIprpcFundIsMinCostCovered(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	spec := ts.Spec("mock")
	err := keeper.NewPendingIprpcFund(ctx, "creator", spec.Index, 1, sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())))
	require.NoError(t, err)
	minCost := keeper.GetMinIprpcCost(ctx)
	latest := keeper.GetLatestPendingIprpcFund(ctx)
	latest.CostCovered = minCost
	require.True(t, keeper.IsIprpcMinCostCovered(ctx, latest))
	latest.CostCovered = minCost.SubAmount(math.OneInt())
	require.False(t, keeper.IsIprpcMinCostCovered(ctx, latest))
}
