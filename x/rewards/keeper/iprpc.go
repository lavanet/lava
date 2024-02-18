package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k Keeper) FundIprpc(ctx sdk.Context, creator string, duration uint64, fund sdk.Coins, spec string) error {
	// verify spec exists and active
	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, spec)
	if !foundAndActive {
		return utils.LavaFormatWarning("spec not found or disabled", types.ErrFundIprpc)
	}

	// check fund consists of minimum amount of ulava (duration * min_iprpc_cost)
	minIprpcFundCost := k.GetMinIprpcCost(ctx).Amount.MulRaw(int64(duration))
	if fund.AmountOf(k.stakingKeeper.BondDenom(ctx)).LT(minIprpcFundCost) {
		return utils.LavaFormatWarning("insufficient ulava tokens in fund. should be at least min iprpc cost * duration", types.ErrFundIprpc,
			utils.LogAttr("min_iprpc_cost", k.GetMinIprpcCost(ctx).String()),
			utils.LogAttr("duration", strconv.FormatUint(duration, 10)),
			utils.LogAttr("fund_ulava_amount", fund.AmountOf(k.stakingKeeper.BondDenom(ctx))),
		)
	}

	// check creator has enough balance
	addr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid creator address", types.ErrFundIprpc)
	}
	creatorUlavaBalance := k.bankKeeper.GetBalance(ctx, addr, k.stakingKeeper.BondDenom(ctx))
	if creatorUlavaBalance.Amount.LT(minIprpcFundCost) {
		return utils.LavaFormatWarning("insufficient ulava tokens in fund. should be at least min iprpc cost * duration", types.ErrFundIprpc,
			utils.LogAttr("min_iprpc_cost", k.GetMinIprpcCost(ctx).String()),
			utils.LogAttr("duration", strconv.FormatUint(duration, 10)),
			utils.LogAttr("creator_ulava_balance", creatorUlavaBalance.String()),
		)
	}

	// send the minimum cost to the validators allocation pool (and subtract them from the fund)
	minIprpcFundCostCoins := sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), minIprpcFundCost))
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, string(types.ValidatorsRewardsAllocationPoolName), minIprpcFundCostCoins)
	if err != nil {
		return utils.LavaFormatError(types.ErrFundIprpc.Error()+"for funding validator allocation pool", err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("min_iprpc_fund_cost", minIprpcFundCost.String()),
		)
	}
	fund = fund.Sub(minIprpcFundCostCoins...)

	// send the funds to the iprpc pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, string(types.IprpcPoolName), fund)
	if err != nil {
		return utils.LavaFormatError(types.ErrFundIprpc.Error()+"for funding iprpc pool", err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("fund", fund.String()),
		)
	}

	// add spec funds to next month IPRPC reward object
	k.addSpecFunds(ctx, spec, fund, duration)

	return nil
}
