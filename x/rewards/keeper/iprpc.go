package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

func (k Keeper) FundIprpc(ctx sdk.Context, creator string, duration uint64, fund sdk.Coins, spec string) error {
	// verify spec exists and active
	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, spec)
	if !foundAndActive {
		return utils.LavaFormatWarning("spec not found or disabled", types.ErrFundIprpc)
	}

	// check fund consists of minimum amount of ulava (min_iprpc_cost)
	minIprpcFundCost := k.GetMinIprpcCost(ctx)
	if fund.AmountOf(k.stakingKeeper.BondDenom(ctx)).LT(minIprpcFundCost.Amount) {
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

	// send the minimum cost to the validators allocation pool (and subtract them from the fund)
	minIprpcFundCostCoins := sdk.NewCoins(minIprpcFundCost).MulInt(sdk.NewIntFromUint64(duration))
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, string(types.ValidatorsRewardsAllocationPoolName), minIprpcFundCostCoins)
	if err != nil {
		return utils.LavaFormatError(types.ErrFundIprpc.Error()+"for funding validator allocation pool", err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("min_iprpc_fund_cost", minIprpcFundCost.String()),
		)
	}
	fund = fund.Sub(minIprpcFundCost)
	allFunds := fund.MulInt(math.NewIntFromUint64(duration))

	// send the funds to the iprpc pool
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, string(types.IprpcPoolName), allFunds)
	if err != nil {
		return utils.LavaFormatError(types.ErrFundIprpc.Error()+"for funding iprpc pool", err,
			utils.LogAttr("creator", creator),
			utils.LogAttr("fund", fund.String()),
		)
	}

	// add spec funds to next month IPRPC reward object
	k.addSpecFunds(ctx, spec, fund, duration, true)

	return nil
}

// handleNoIprpcRewardToProviders handles the situation in which there are no providers to send IPRPC rewards to
// so the IPRPC rewards transfer to the next month
func (k Keeper) handleNoIprpcRewardToProviders(ctx sdk.Context, iprpcFunds []types.Specfund) {
	for _, fund := range iprpcFunds {
		k.addSpecFunds(ctx, fund.Spec, fund.Fund, 1, false)
	}

	details := map[string]string{
		"transferred_funds": fmt.Sprint(iprpcFunds),
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.TransferIprpcRewardToNextMonthEventName, details,
		"No provider serviced an IPRPC eligible subscription, transferring current month IPRPC funds to next month")
}

// countIprpcCu updates the specCuMap which keeps records of spec->SpecCuType (which holds the IPRPC CU per provider)
func (k Keeper) countIprpcCu(specCuMap map[string]types.SpecCuType, iprpcCu uint64, spec string, provider string) {
	if iprpcCu != 0 {
		specCu, ok := specCuMap[spec]
		if !ok {
			specCuMap[spec] = types.SpecCuType{
				ProvidersCu: []types.ProviderCuType{{Provider: provider, CU: iprpcCu}},
				TotalCu:     iprpcCu,
			}
		} else {
			specCu.ProvidersCu = append(specCu.ProvidersCu, types.ProviderCuType{Provider: provider, CU: iprpcCu})
			specCu.TotalCu += iprpcCu
			specCuMap[spec] = specCu
		}
	}
}

// AddSpecFunds adds funds for a specific spec for <duration> of months.
// This function is used by the fund-iprpc TX.
// use fromNextMonth=true for normal IPRPC fund (should always start from next month)
// use fromNextMonth=false for IPRPC reward transfer for next month (when no providers are eligible for IPRPC rewards)
func (k Keeper) addSpecFunds(ctx sdk.Context, spec string, fund sdk.Coins, duration uint64, fromNextMonth bool) {
	startID := k.GetIprpcRewardsCurrentId(ctx)
	if fromNextMonth {
		startID++ // fund IPRPC only from the next month for <duration> months
	}

	for i := startID; i < startID+duration; i++ {
		iprpcReward, found := k.GetIprpcReward(ctx, i)
		if found {
			// found IPRPC reward, find if spec exists
			specFound := false
			for i := 0; i < len(iprpcReward.SpecFunds); i++ {
				if iprpcReward.SpecFunds[i].Spec == spec {
					specFound = true
					iprpcReward.SpecFunds[i].Fund = iprpcReward.SpecFunds[i].Fund.Add(fund...)
					break
				}
			}
			if !specFound {
				iprpcReward.SpecFunds = append(iprpcReward.SpecFunds, types.Specfund{Spec: spec, Fund: fund})
			}
		} else {
			// did not find IPRPC reward -> create a new one
			iprpcReward.Id = i
			iprpcReward.SpecFunds = []types.Specfund{{Spec: spec, Fund: fund}}
		}
		k.SetIprpcReward(ctx, iprpcReward)
	}
}

// distributeIprpcRewards is distributing the IPRPC rewards for providers according to their serviced CU
func (k Keeper) distributeIprpcRewards(ctx sdk.Context, iprpcReward types.IprpcReward, specCuMap map[string]types.SpecCuType) {
	// none of the providers will get the IPRPC reward this month, transfer the funds to the next month
	if len(specCuMap) == 0 {
		k.handleNoIprpcRewardToProviders(ctx, iprpcReward.SpecFunds)
		return
	}

	leftovers := sdk.NewCoins()
	for _, specFund := range iprpcReward.SpecFunds {
		details := map[string]string{}

		// verify specCuMap holds an entry for the relevant spec
		specCu, ok := specCuMap[specFund.Spec]
		if !ok {
			k.handleNoIprpcRewardToProviders(ctx, []types.Specfund{specFund})
			utils.LavaFormatError("did not distribute iprpc rewards to providers in spec", fmt.Errorf("specCU not found"),
				utils.LogAttr("spec", specFund.Spec),
				utils.LogAttr("rewards", specFund.Fund.String()),
			)
			continue
		}

		// tax the rewards to the community and validators
		fundAfterTax := sdk.NewCoins()
		for _, coin := range specFund.Fund {
			leftover, err := k.ContributeToValidatorsAndCommunityPool(ctx, coin, string(types.IprpcPoolName))
			if err != nil {
				// if ContributeToValidatorsAndCommunityPool fails we continue with the next providerrewards
				continue
			}
			fundAfterTax = fundAfterTax.Add(leftover)
		}
		specFund.Fund = fundAfterTax

		UsedReward := sdk.NewCoins()
		// distribute IPRPC reward for spec
		for _, providerCU := range specCu.ProvidersCu {
			if specCu.TotalCu == 0 {
				// spec was not serviced by any provider, continue
				continue
			}
			// calculate provider IPRPC reward
			providerIprpcReward := specFund.Fund.MulInt(sdk.NewIntFromUint64(providerCU.CU)).QuoInt(sdk.NewIntFromUint64(specCu.TotalCu))

			UsedRewardTemp := UsedReward.Add(providerIprpcReward...)
			if UsedReward.IsAnyGT(specFund.Fund) {
				utils.LavaFormatError("failed to send iprpc rewards to provider", fmt.Errorf("tried to send more rewards than funded"), utils.LogAttr("provider", providerCU))
				break
			}
			UsedReward = UsedRewardTemp

			// reward the provider
			_, _, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerCU.Provider, specFund.Spec, providerIprpcReward, string(types.IprpcPoolName), false, false, false)
			if err != nil {
				utils.LavaFormatError("failed to send iprpc rewards to provider", err, utils.LogAttr("provider", providerCU))
			}
			details[providerCU.Provider] = fmt.Sprintf("cu: %d reward: %s", providerCU.CU, providerIprpcReward.String())
		}
		details["total_cu"] = strconv.FormatUint(specCu.TotalCu, 10)
		details["total_reward"] = specFund.Fund.String()
		details["chainid"] = specFund.GetSpec()
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.IprpcPoolEmissionEventName, details, "IPRPC monthly rewards distributed successfully")

		// count used rewards
		leftovers = leftovers.Add(specFund.Fund.Sub(UsedReward...)...)
	}

	// handle leftovers
	err := k.FundCommunityPoolFromModule(ctx, leftovers, string(types.IprpcPoolName))
	if err != nil {
		utils.LavaFormatError("could not send iprpc leftover to community pool", err)
	}
}
