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
	"github.com/lavanet/lava/testutil/sample"
	commontypes "github.com/lavanet/lava/utils/common/types"
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
		`{"client":"bruce","duration":2}`,
		`{"iprpc":{"creator":"my-moniker","duration":2,"spec":"mockspec"}}`,
		`{"iprpc":{"creator":"","duration":2,"spec":"mockspec"}}`,
		`{"iprpc":{"creator":"mockspec","duration":2,"spec":"mockspec"}}`,
		`{"iprpc":{"creator":"mockspec","duration":2,"spec":"mockspec"}}`,
		`{"iprpc":{"creator":"my-moniker","duration":2,"spec":"other-mockspec"}}`,
		`{"iprpc":{"creator":"my-moniker","duration":2}}`,
		`{"iprpc":{"creator":"my-moniker","duration":-2,"spec":"mockspec"}}`,
		`{"iprpc":{"creator":"my-moniker","spec":"mockspec"}}`,
	}

	const (
		EMPTY = iota
		NOT_JSON
		JSON_NO_IPRPC
		VALID_JSON_IPRPC
		EMPTY_CREATOR_JSON_IPRPC
		CREATOR_IS_SPEC_JSON_IPRPC
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
			expectedMemo: types.IprpcMemo{Creator: "my-moniker", Spec: "mockspec", Duration: 2},
		},
		{
			name:         "invalid memo iprpc json - invalid creator - empty creator",
			memoInd:      EMPTY_CREATOR_JSON_IPRPC,
			expectError:  types.ErrIprpcMemoInvalid,
			expectedMemo: types.IprpcMemo{},
		},
		{
			name:         "invalid memo iprpc json - invalid creator - creator is named like on-chain spec",
			memoInd:      CREATOR_IS_SPEC_JSON_IPRPC,
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

// createNPendingIbcIprpcFunds is a helper function that creates an n-sized array of PendingIbcIprpcFund objects
func createNPendingIbcIprpcFunds(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PendingIbcIprpcFund {
	items := make([]types.PendingIbcIprpcFund, n)
	for i := range items {
		items[i] = types.PendingIbcIprpcFund{
			Index:    uint64(i),
			Creator:  "dummy",
			Spec:     "mock",
			Duration: uint64(i),
			Fund:     GetIbcCoins(sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(int64(i+1)))),
			Expiry:   uint64(ctx.BlockTime().UTC().Unix()) + uint64(i),
		}
		keeper.SetPendingIbcIprpcFund(ctx, items[i])
	}
	return items
}

// TestPendingIbcIprpcFundsGet tests GetPendingIbcIprpcFund()
func TestPendingIbcIprpcFundsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		res, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		require.True(t, found)
		require.True(t, res.IsEqual(item))
	}
}

// TestPendingIbcIprpcFundsRemove tests RemovePendingIbcIprpcFund
func TestPendingIbcIprpcFundsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePendingIbcIprpcFund(ctx, item.Index)
		_, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		require.False(t, found)
	}
}

// TestPendingIbcIprpcFundsGetAll tests GetAllPendingIbcIprpcFund
func TestPendingIbcIprpcFundsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPendingIbcIprpcFund(ctx)),
	)
}

// TestPendingIbcIprpcFundsRemoveExpired tests RemoveExpiredPendingIbcIprpcFunds and IsExpired
func TestPendingIbcIprpcFundsRemoveExpired(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)

	// advance time so some of the PendingIbcIprpcFund will expire
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(3 * time.Second))

	// verify they're expired
	for i := range items {
		if i <= 3 {
			require.True(t, items[i].IsExpired(ctx))
		} else {
			require.False(t, items[i].IsExpired(ctx))
		}
	}

	// remove expired PendingIbcIprpcFund and check they cannot be found
	keeper.RemoveExpiredPendingIbcIprpcFunds(ctx)
	for _, item := range items {
		_, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		if item.Index <= 3 {
			require.False(t, found)
		} else {
			require.True(t, found)
		}
	}
}

// TestPendingIbcIprpcFundsRemoveExpiredWithBeginBlock tests that expired PendingIbcIprpcFunds are removed with BeginBlock
// Also, their funds should be sent to the community pool
func TestPendingIbcIprpcFundsRemoveExpiredWithBeginBlock(t *testing.T) {
	ts := newTester(t, false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	items := createNPendingIbcIprpcFunds(&keeper, ctx, 10)

	// let funder be the account that sends the ibc-transfer msg
	funder := sample.AccAddressObject()
	funderBalance := sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(10000)))
	err := ts.Keepers.BankKeeper.SetBalance(ctx, funder, funderBalance)
	require.NoError(t, err)

	// IbcIprpcReceiver gets its balance via an IBC transfer
	ts.SendIprpcOverIbcTransferPacket(funder, sdk.NewCoin(ts.TokenDenom(), iprpcFunds.AmountOf(ts.TokenDenom())), 1)

	// advance block with 3 seconds to expire some of the PendingIbcIprpcFunds
	ts.AdvanceBlock(3 * time.Second)

	// check that expired PendingIbcIprpcFunds were removed
	for _, item := range items {
		_, found := keeper.GetPendingIbcIprpcFund(ctx, item.Index)
		if item.Index <= 3 {
			require.False(t, found)
		} else {
			require.True(t, found)
		}
	}

	// check the community pool's balance (objects in indices 0-3 were removed, so expected balance is 1+2+3+4=10ulava)
	expectedBalance := GetIbcCoins(sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(10)))
	communityCoins := ts.Keepers.Distribution.GetFeePoolCommunityCoins(ts.Ctx)
	communityBalance := communityCoins.AmountOf(expectedBalance.Denom).TruncateInt()
	require.True(t, communityBalance.Equal(expectedBalance.Amount))
}

// TestPendingIbcIprpcFundGetLatest tests GetLatestPendingIbcIprpcFund
func TestPendingIbcIprpcFundGetLatest(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	latest := keeper.GetLatestPendingIbcIprpcFund(ctx)
	require.True(t, latest.IsEmpty())
	items := createNPendingIbcIprpcFunds(keeper, ctx, 10)
	latest = keeper.GetLatestPendingIbcIprpcFund(ctx)
	require.True(t, latest.IsEqual(items[len(items)-1]))
}

// TestPendingIbcIprpcFundNew tests NewPendingIbcIprpcFund
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
		{"invalid fund", spec.Index, sdk.NewCoin(ts.TokenDenom(), math.ZeroInt()), false},
		{"non-existent spec", "eth", validFunds, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := keeper.NewPendingIbcIprpcFund(ctx, "creator", tt.spec, 1, tt.funds)
			if tt.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestCalcPendingIbcIprpcFundMinCost tests CalcPendingIbcIprpcFundMinCost
func TestCalcPendingIbcIprpcFundMinCost(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	latest := keeper.GetLatestPendingIbcIprpcFund(ctx)
	minCost := keeper.CalcPendingIbcIprpcFundMinCost(ctx, latest)
	expectedMinCost := sdk.NewCoin(ts.TokenDenom(), keeper.GetMinIprpcCost(ctx).Amount.MulRaw(int64(latest.Duration)))
	require.True(t, minCost.IsEqual(expectedMinCost))
}

// TestCalcPendingIbcIprpcFundExpiration tests CalcPendingIbcIprpcFundExpiration
func TestCalcPendingIbcIprpcFundExpiration(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	expectedExpiry := uint64(ctx.BlockTime().Add(keeper.IbcIprpcExpiration(ctx)).UTC().Unix())
	expiry := keeper.CalcPendingIbcIprpcFundExpiration(ctx)
	require.Equal(t, expectedExpiry, expiry)
}

// TestPendingIbcIprpcFundsQuery tests that the pending-ibc-iprpc-funds query works as expected with filters
func TestPendingIbcIprpcFundsQuery(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx
	items := createNPendingIbcIprpcFunds(&keeper, ctx, 3)

	// make some of the PendingIbcIprpcFunds different with creator and spec
	items[0].Creator = "blabla"
	items[1].Spec = mockSpec2
	keeper.SetPendingIbcIprpcFund(ctx, items[0])
	keeper.SetPendingIbcIprpcFund(ctx, items[1])

	minCost := keeper.GetMinIprpcCost(ctx)
	template := []struct {
		name                            string
		filter                          string
		expectedPendingIbcIprpcFundInfo []types.PendingIbcIprpcFundInfo
		success                         bool
	}{
		{"no filter", "", []types.PendingIbcIprpcFundInfo{
			{PendingIbcIprpcFund: items[0], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[0].Duration)))},
			{PendingIbcIprpcFund: items[1], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[1].Duration)))},
			{PendingIbcIprpcFund: items[2], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[2].Duration)))},
		}, true},
		{"index filter", "2", []types.PendingIbcIprpcFundInfo{
			{PendingIbcIprpcFund: items[2], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[2].Duration)))},
		}, true},
		{"creator filter", "blabla", []types.PendingIbcIprpcFundInfo{
			{PendingIbcIprpcFund: items[0], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[0].Duration)))},
		}, true},
		{"spec filter", mockSpec2, []types.PendingIbcIprpcFundInfo{
			{PendingIbcIprpcFund: items[1], Cost: sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(int64(items[1].Duration)))},
		}, true},
		{"invalid index filter", "100", []types.PendingIbcIprpcFundInfo{}, false},
		{"invalid creator/spec filter", "yoyo", []types.PendingIbcIprpcFundInfo{}, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ts.QueryRewardsPendingIbcIprpcFunds(tt.filter)
			if tt.success {
				require.NoError(t, err)
				foundMatch := false
				for _, piifi := range res.PendingIbcIprpcFundsInfo {
					for _, expectedPiifi := range tt.expectedPendingIbcIprpcFundInfo {
						if piifi.PendingIbcIprpcFund.IsEqual(expectedPiifi.PendingIbcIprpcFund) && piifi.Cost.IsEqual(expectedPiifi.Cost) {
							foundMatch = true
							break
						}
					}
					if !foundMatch {
						require.FailNow(t, "info result not matching expected")
					}
					foundMatch = false
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestPendingIbcIprpcFundNewFunds tests that when creating a new PendingIbcIprpcFund the original
// fund gets divided by duration and the division leftovers are transferred to the community pool
func TestPendingIbcIprpcFundNewFunds(t *testing.T) {
	template := []struct {
		name                     string
		funds                    math.Int
		duration                 uint64
		expectedFundsInPending   math.Int
		expectedFundsInCommunity math.Int
		success                  bool
	}{
		{"divisiable - 9ulava", math.NewInt(9), 3, math.NewInt(3), math.ZeroInt(), true},
		{"not divisiable - 10ulava", math.NewInt(10), 3, math.NewInt(3), math.OneInt(), true},
		{"less than duration - 1ulava", math.NewInt(1), 3, math.ZeroInt(), math.ZeroInt(), false},
		{"one month duration - 10ulava", math.NewInt(10), 1, math.NewInt(10), math.ZeroInt(), true},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTester(t, false)
			keeper, ctx := ts.Keepers.Rewards, ts.Ctx
			spec := ts.Spec("mock")
			funds := sdk.NewCoin(ts.TokenDenom(), tt.funds)

			// let funder be the account that sends the ibc-transfer msg
			funder := sample.AccAddressObject()
			funderBalance := sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(10000)))
			err := ts.Keepers.BankKeeper.SetBalance(ctx, funder, funderBalance)
			require.NoError(t, err)

			// IbcIprpcReceiver gets its balance via an IBC transfer (leftover funds are taken from it to the community pool)
			ts.SendIprpcOverIbcTransferPacket(funder, funds, 1)

			// create a new PendingIbcIprpcFund
			piif, leftovers, err := keeper.NewPendingIbcIprpcFund(ctx, "creator", spec.Index, tt.duration, funds)
			if tt.success {
				require.NoError(t, err)
				require.True(t, piif.Fund.Amount.Equal(tt.expectedFundsInPending))
				require.True(t, leftovers.Amount.Equal(tt.expectedFundsInCommunity))
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestCoverIbcIprpcFundCost tests that the cover-ibc-iprpc-fund-cost transaction
// Scenarios:
//  0. Create 2 PendingIbcIprpcFund objects and fund PendingIprpcPool and gov. First with 101ulava for 2 months, second with
//     99ulava for 2 months. Expected: PendingIbcIprpcFund with 50ulava, PendingIbcIprpcFund with 49ulava, community pool 2ulava
//  1. Cover costs with alice for first PendingIbcIprpcFund. Expect two iprpc rewards from next month of 50ulava, PendingIbcIprpcFund
//     removed, IPRPC pool with 100ulava, second PendingIbcIprpcFund remains (49ulava), alice balance reduced by MinIprpcCost
//  2. Cover costs with gov module for second PendingIbcIprpcFund. Expect two iprpc rewards from next month of 99ulava, PendingIbcIprpcFund
//     removed, IPRPC pool with 198ulava, gov module balance not reduced by MinIprpcCost
func TestCoverIbcIprpcFundCost(t *testing.T) {
	ts := newTester(t, true)
	ts.setupForIprpcTests(false)
	keeper, ctx := ts.Keepers.Rewards, ts.Ctx

	// let funder be a dummy account to send the IBC transfer coins
	// let alice be the account that cover costs
	funder := sample.AccAddressObject()
	funderBalance := sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(10000)))
	err := ts.Keepers.BankKeeper.SetBalance(ctx, funder, funderBalance)
	require.NoError(t, err)

	alice := sample.AccAddressObject()
	aliceBalance := sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(10000)))
	err = ts.Keepers.BankKeeper.SetBalance(ctx, alice, aliceBalance)
	require.NoError(t, err)

	// set min IPRPC cost to be 50ulava (for validation checks later)
	minCost := sdk.NewCoin(ts.TokenDenom(), math.NewInt(50))
	keeper.SetMinIprpcCost(ctx, minCost)

	// create 2 pending IPRPC requests
	funds1 := sdk.NewCoin(ts.TokenDenom(), math.NewInt(101)) // will be index 0
	ts.SendIprpcOverIbcTransferPacket(funder, funds1, 2)
	funds2 := sdk.NewCoin(ts.TokenDenom(), math.NewInt(99)) // will be index 1
	ts.SendIprpcOverIbcTransferPacket(funder, funds2, 2)
	expectedPendingIprpcPoolBalance := GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(198))) // 99+101-leftovers = 99+101-2
	pendingIprpcPoolBalance := ts.Keepers.Rewards.TotalPoolTokens(ctx, types.PendingIprpcPoolName)
	require.True(t, pendingIprpcPoolBalance.IsEqual(sdk.NewCoins(expectedPendingIprpcPoolBalance)))

	// fund the gov module with a dummy balance, just to see it's not changing
	govModuleBalance := sdk.NewCoins(sdk.NewCoin(ts.TokenDenom(), math.OneInt()))
	govModule := ts.Keepers.AccountKeeper.GetModuleAddress("gov")
	err = ts.Keepers.BankKeeper.SetBalance(ctx, govModule, govModuleBalance)
	require.NoError(t, err)

	// cover costs of first PendingIbcIprpcFund with alice
	expectedMinCost := sdk.NewCoin(minCost.Denom, minCost.Amount.MulRaw(2))
	_, err = ts.TxRewardsCoverIbcIprpcFundCost(alice.String(), 0)
	require.NoError(t, err)
	_, found := keeper.GetPendingIbcIprpcFund(ctx, 0)
	require.False(t, found)
	require.Equal(t, expectedMinCost.Amount.Int64(), aliceBalance.AmountOf(ts.TokenDenom()).Int64()-ts.GetBalance(alice))

	res, err := ts.QueryRewardsIprpcSpecReward(mockSpec)
	require.NoError(t, err)
	expectedIprpcRewards := []types.IprpcReward{
		{Id: 1, SpecFunds: []types.Specfund{{Spec: mockSpec, Fund: sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(50))))}}},
		{Id: 2, SpecFunds: []types.Specfund{{Spec: mockSpec, Fund: sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(50))))}}},
	}
	require.Len(t, res.IprpcRewards, len(expectedIprpcRewards))
	for i := range res.IprpcRewards {
		require.Equal(t, expectedIprpcRewards[i].Id, res.IprpcRewards[i].Id)
		require.ElementsMatch(t, expectedIprpcRewards[i].SpecFunds, res.IprpcRewards[i].SpecFunds)
	}

	_, found = keeper.GetPendingIbcIprpcFund(ctx, 1)
	require.True(t, found)

	expectedIprpcPoolBalance := sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(100))))
	iprpcPoolBalance := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.IprpcPoolName)
	require.True(t, expectedIprpcPoolBalance.IsEqual(iprpcPoolBalance))

	// cover costs of second PendingIbcIprpcFund with gov
	// note that the gov module's balance should not change since it's the only account
	// that doesn't need to pay min cost (see check below)
	_, err = ts.TxRewardsCoverIbcIprpcFundCost(govModule.String(), 1)
	require.NoError(t, err)
	_, found = keeper.GetPendingIbcIprpcFund(ctx, 1)
	require.False(t, found)
	require.Equal(t, int64(0), govModuleBalance.AmountOf(ts.TokenDenom()).Int64()-ts.GetBalance(govModule))

	res, err = ts.QueryRewardsIprpcSpecReward(mockSpec)
	require.NoError(t, err)
	expectedIprpcRewards = []types.IprpcReward{
		{Id: 1, SpecFunds: []types.Specfund{{Spec: mockSpec, Fund: sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(99))))}}},
		{Id: 2, SpecFunds: []types.Specfund{{Spec: mockSpec, Fund: sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(99))))}}},
	}
	require.Len(t, res.IprpcRewards, len(expectedIprpcRewards))
	for i := range res.IprpcRewards {
		require.Equal(t, expectedIprpcRewards[i].Id, res.IprpcRewards[i].Id)
		require.ElementsMatch(t, expectedIprpcRewards[i].SpecFunds, res.IprpcRewards[i].SpecFunds)
	}

	expectedIprpcPoolBalance = sdk.NewCoins(GetIbcCoins(sdk.NewCoin(ts.TokenDenom(), math.NewInt(198))))
	iprpcPoolBalance = ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.IprpcPoolName)
	require.True(t, expectedIprpcPoolBalance.IsEqual(iprpcPoolBalance))

	// verify that PendingIprpcPool has 0ulava balance (in the IBC middleware, the leftovers
	// of both IPRPC over IBC are sent to the community pool)
	pendingIprpcPoolBalance = ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, types.PendingIprpcPoolName)
	require.True(t, pendingIprpcPoolBalance.Empty())
}
