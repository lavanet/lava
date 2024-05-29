package keeper_test

import (
	"testing"

	"strconv"
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
		    "duration": "3"	
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": "3"
			}
		}`,
		`{
			"iprpc": {
			  "creator": "",
			  "spec": "mockspec",
			  "duration": "3"
			}
		}`,
		`{
			"iprpc": {
			  "spec": "mockspec",
			  "duration": "3"
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "other-mockspec",
			  "duration": "3"
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "duration": "3"
			}
		}`,
		`{
			"iprpc": {
			  "creator": "my-moniker",
			  "spec": "mockspec",
			  "duration": "-3"
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

func createNPendingIbcIprpcFunds(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PendingIbcIprpcFund {
	items := make([]types.PendingIbcIprpcFund, n)
	for i := range items {
		items[i] = types.PendingIbcIprpcFund{
			Index:    uint64(i),
			Creator:  "dummy",
			Spec:     "mock",
			Duration: uint64(i),
			Expiry:   uint64(ctx.BlockTime().UTC().Unix()) + uint64(i),
		}
		keeper.SetPendingIbcIprpcFund(ctx, items[i])
	}
	return items
}

func TestPendingIbcIprpcFundsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		res, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		require.True(t, found)
		require.True(t, res.IsEqual(item))
	}
}

func TestPendingIbcIprpcFundsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePendingIbcIprpcFund(ctx, item.Index)
		_, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		require.False(t, found)
	}
}

func TestPendingIbcIprpcFundsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPendingIbcIprpcFund(ctx)),
	)
}

func TestPendingIbcIprpcFundsRemoveExpired(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(3 * time.Second))
	keeper.RemoveExpiredPendingIbcIprpcFund(ctx)
	for _, item := range items {
		_, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		if item.Index <= 3 {
			require.False(t, found)
		} else {
			require.True(t, found)
		}
	}
}

func TestPendingIbcIprpcFundGetLatest(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	latest := keeper.GetLatestPendingIbcIprpcFund(ctx)
	require.True(t, latest.IsEmpty())
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	latest = keeper.GetLatestPendingIbcIprpcFund(ctx)
	require.True(t, latest.IsEqual(items[len(items)-1]))
}

func TestPendingIbcIprpcFundNew(t *testing.T) {
	ts := newTester(t, false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	spec := ts.Spec("mock")
	validFunds := sdk.NewCoin("denom", math.OneInt())

	template := []struct {
		name    string
		spec    string
		funds   sdk.Coin
		success bool
	}{
		{"valid", spec.Index, validFunds, true},
		{"invalid fund", spec.Index, sdk.NewCoin("", math.NewInt(-1)), false},
		{"non-existent spec", "eth", validFunds, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			err := keeper.NewPendingIbcIprpcFund(ctx, "creator", tt.spec, 1, tt.funds)
			if tt.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPendingIbcIprpcFundIsMinCostCovered(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	latest := keeper.GetLatestPendingIbcIprpcFund(ctx)
	minCost := keeper.CalcPendingIbcIprpcFundMinCost(ctx, latest)
	expectedMinCost := sdk.NewCoin(ts.TokenDenom(), keeper.GetMinIprpcCost(ctx).Amount.MulRaw(int64(latest.Duration)))
	require.True(t, minCost.IsEqual(expectedMinCost))
}
