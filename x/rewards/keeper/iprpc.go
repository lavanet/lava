package keeper

import (
	"fmt"
	"sort"
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

// handleNoIprpcRewardToProviders handles the situation in which there are no providers to send IPRPC rewards to
// so the IPRPC rewards transfer to the next month
func (k Keeper) handleNoIprpcRewardToProviders(ctx sdk.Context, iprpcReward types.IprpcReward) {
	nextMonthIprpcReward, found := k.PopIprpcReward(ctx, false)
	nextMonthId := k.GetIprpcRewardsCurrent(ctx)
	if !found {
		nextMonthIprpcReward = types.IprpcReward{Id: nextMonthId, SpecFunds: iprpcReward.SpecFunds}
	} else {
		nextMonthIprpcReward.SpecFunds = k.transferSpecFundsToNextMonth(iprpcReward.SpecFunds, nextMonthIprpcReward.SpecFunds)
	}
	k.SetIprpcReward(ctx, nextMonthIprpcReward)
	details := map[string]string{
		"transferred_funds":        iprpcReward.String(),
		"next_month_updated_funds": nextMonthIprpcReward.String(),
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.TransferIprpcRewardToNextMonth, details,
		"No provider serviced an IPRPC eligible subscription, transferring current month IPRPC funds to next month")
}

// countIprpcCu updates the specCuMap which keeps records of spec->SpecCuType (which holds the IPRPC CU per provider)
func (k Keeper) countIprpcCu(specCuMap map[string]types.SpecCuType, iprpcCu uint64, spec string, provider string) {
	if iprpcCu != 0 {
		specCu, ok := specCuMap[spec]
		if !ok {
			specCuMap[spec] = types.SpecCuType{
				ProvidersCu: map[string]uint64{provider: iprpcCu},
				TotalCu:     iprpcCu,
			}
		} else {
			_, ok := specCu.ProvidersCu[provider]
			if !ok {
				specCu.ProvidersCu[provider] = iprpcCu
			} else {
				specCu.ProvidersCu[provider] += iprpcCu
			}
			specCu.TotalCu += iprpcCu
			specCuMap[spec] = specCu
		}
	}
}

// AddSpecFunds adds funds for a specific spec for <duration> of months.
// This function is used by the fund-iprpc TX.
func (k Keeper) addSpecFunds(ctx sdk.Context, spec string, fund sdk.Coins, duration uint64) {
	startID := k.GetIprpcRewardsCurrent(ctx) + 1 // fund IPRPC only from the next month for <duration> months
	for i := startID; i < startID+duration; i++ {
		iprpcReward, found := k.GetIprpcReward(ctx, i)
		if found {
			// found IPRPC reward, find if spec exists
			specFound := false
			for i := 0; i < len(iprpcReward.SpecFunds); i++ {
				if iprpcReward.SpecFunds[i].Spec == spec {
					specFound = true
					iprpcReward.SpecFunds[i].Fund = iprpcReward.SpecFunds[i].Fund.Add(fund...)
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

// transferSpecFundsToNextMonth transfer the specFunds to the next month's IPRPC funds
// this function is used when there are no providers that should get the monthly IPRPC reward,
// so the reward transfers to the next month
func (k Keeper) transferSpecFundsToNextMonth(specFunds []types.Specfund, nextMonthSpecFunds []types.Specfund) []types.Specfund {
	// Create a slice to store merged spec funds
	var mergedList []types.Specfund

	// Loop through current spec funds
	for _, current := range specFunds {
		found := false

		// Loop through next month spec funds
		for i, next := range nextMonthSpecFunds {
			// If the spec is found in next month spec funds, merge the funds
			if current.Spec == next.Spec {
				// Add current month's fund to next month's fund
				nextMonthSpecFunds[i].Fund = nextMonthSpecFunds[i].Fund.Add(current.Fund...)
				found = true
				break
			}
		}

		// If spec is not found in next month spec funds, add it to the merged list
		if !found {
			mergedList = append(mergedList, current)
		}
	}

	// Append any remaining spec funds from next month that were not merged
	mergedList = append(mergedList, nextMonthSpecFunds...)

	// Sort the merged list by spec
	sort.Slice(mergedList, func(i, j int) bool { return mergedList[i].Spec < mergedList[j].Spec })

	return mergedList
}

// distributeIprpcRewards is distributing the IPRPC rewards for providers according to their serviced CU
func (k Keeper) distributeIprpcRewards(ctx sdk.Context, iprpcReward types.IprpcReward, specCuMap map[string]types.SpecCuType) {
	usedReward := sdk.NewCoins()
	for _, specFund := range iprpcReward.SpecFunds {
		// verify specCuMap holds an entry for the relevant spec
		specCu, ok := specCuMap[specFund.Spec]
		if !ok {
			utils.LavaFormatError("did not distribute iprpc rewards to providers in spec", fmt.Errorf("specCU not found"),
				utils.LogAttr("spec", specFund.Spec),
				utils.LogAttr("rewards", specFund.Fund.String()),
			)
			continue
		}

		// collect providers details
		providers := []string{}
		for provider := range specCu.ProvidersCu {
			providers = append(providers, provider)
		}
		sort.Strings(providers)

		// distribute IPRPC reward for spec
		for _, provider := range providers {
			providerAddr, err := sdk.AccAddressFromBech32(provider)
			if err != nil {
				continue
			}
			if specCu.TotalCu == 0 {
				// spec was not serviced by any provider, continue
				continue
			}
			// calculate provider IPRPC reward
			providerIprpcReward := specFund.Fund.MulInt(sdk.NewIntFromUint64(specCu.ProvidersCu[provider])).QuoInt(sdk.NewIntFromUint64(specCu.TotalCu))

			// reward the provider
			_, _, err = k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, specFund.Spec, providerIprpcReward, string(types.IprpcPoolName), false, false, false)
			if err != nil {
				utils.LavaFormatError("failed to send iprpc rewards to provider", err, utils.LogAttr("provider", provider))
			}

			usedReward = usedReward.Add(providerIprpcReward...)
		}

		// count used rewards
		usedReward = specFund.Fund.Sub(usedReward...)
	}

	// handle leftovers
	err := k.FundCommunityPoolFromModule(ctx, usedReward, string(types.IprpcPoolName))
	if err != nil {
		utils.LavaFormatError("could not send iprpc leftover to community pool", err)
	}

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.IprpcPoolEmissionEventName, map[string]string{"iprpc_rewards_leftovers": usedReward.String()}, "IPRPC monthly rewards distributed successfully")
}
