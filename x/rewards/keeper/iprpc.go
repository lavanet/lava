package keeper

import (
	"fmt"
	"sort"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

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
			specIndex := -1
			for i := 0; i < len(iprpcReward.SpecFunds); i++ {
				if iprpcReward.SpecFunds[i].Spec == spec {
					specIndex = i
				}
			}
			// update spec funds
			if specIndex >= 0 {
				iprpcReward.SpecFunds[specIndex].Fund = iprpcReward.SpecFunds[specIndex].Fund.Add(fund...)
			} else {
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
	mergedMap := make(map[string]sdk.Coins)

	// populate map with current spec funds
	for i, obj := range specFunds {
		mergedMap[obj.Spec] = specFunds[i].Fund
	}

	// update the merged map with the next month spec funds
	for i, obj := range nextMonthSpecFunds {
		if fund, ok := mergedMap[obj.Spec]; ok {
			mergedMap[obj.Spec] = fund.Add(nextMonthSpecFunds[i].Fund...)
		} else {
			mergedMap[obj.Spec] = obj.Fund
		}
	}

	// Convert map back to list and sort
	var mergedList []types.Specfund
	for spec, fund := range mergedMap {
		mergedList = append(mergedList, types.Specfund{Spec: spec, Fund: fund})
	}
	sort.Slice(mergedList, func(i, j int) bool { return mergedList[i].Spec < mergedList[j].Spec })

	return mergedList
}

// distributeIprpcRewards is distributing the IPRPC rewards for providers according to their serviced CU
func (k Keeper) distributeIprpcRewards(ctx sdk.Context, iprpcReward types.IprpcReward, specCuMap map[string]types.SpecCuType) {
	for _, specFund := range iprpcReward.SpecFunds {
		// collect details
		details := map[string]string{"spec": specFund.Spec}
		rewardsStr := []string{}
		for _, reward := range specFund.Fund {
			rewardsStr = append(rewardsStr, reward.String())
		}
		details["rewards"] = strings.Join(rewardsStr, ",")

		// verify specCuMap holds an entry for the relevant spec
		specCu, ok := specCuMap[specFund.Spec]
		if !ok {
			utils.LavaFormatError("did not distribute iprpc rewards to providers in spec", fmt.Errorf("specCU not found"),
				utils.LogAttr("spec", details["spec"]),
				utils.LogAttr("rewards", details["rewards"]),
			)
			continue
		}

		// collect providers details
		providers := []string{}
		for provider := range specCu.ProvidersCu {
			providers = append(providers, provider)
		}
		sort.Strings(providers)
		details["providers"] = strings.Join(providers, ",")

		// distribute IPRPC reward for spec
		usedReward := sdk.NewCoins()
		for _, provider := range providers {
			providerAddr, err := sdk.AccAddressFromBech32(provider)
			if err != nil {
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

		// handle leftovers
		usedReward = specFund.Fund.Sub(usedReward...)
		err := k.FundCommunityPoolFromModule(ctx, usedReward, string(types.IprpcPoolName))
		if err != nil {
			utils.LavaFormatError("could not send iprpc leftover to community pool", err)
		}

		utils.LogLavaEvent(ctx, k.Logger(ctx), types.IprpcPoolEmissionEventName, details, "IPRPC monthly rewards distributed successfully")
	}
}
