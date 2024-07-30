package keeper

import (
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) UnstakeEntry(ctx sdk.Context, validator, chainID, creator, unstakeDescription string) error {
	logger := k.Logger(ctx)
	// TODO: validate chainID basic validation

	// we can unstake disabled specs, but not missing ones
	_, found := k.specKeeper.GetSpec(ctx, chainID)
	if !found {
		return utils.LavaFormatWarning("trying to unstake an entry on missing spec", fmt.Errorf("spec not found"),
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	if _, err := sdk.ValAddressFromBech32(validator); err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "validator", Value: validator},
		)
	}

	existingEntry, entryExists := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, creator)
	if !entryExists {
		return utils.LavaFormatWarning("can't unstake Entry, stake entry not found for address", fmt.Errorf("stake entry not found"),
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	if creator != existingEntry.Vault {
		return utils.LavaFormatWarning("can't unstake entry with provider address, only vault address is allowed to unstake", fmt.Errorf("provider unstake failed"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("provider", existingEntry.Address),
			utils.LogAttr("vault", existingEntry.Vault),
			utils.LogAttr("chain_id", chainID),
		)
	}

	// the stake entry is removed inside UnbondFull
	err := k.dualstakingKeeper.UnbondFull(ctx, existingEntry.Vault, validator, existingEntry.Address, existingEntry.GetChain(), existingEntry.Stake, true)
	if err != nil {
		return utils.LavaFormatWarning("can't unbond self delegation", err,
			utils.Attribute{Key: "address", Value: existingEntry.Address},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}

	details := map[string]string{
		"address":     existingEntry.GetAddress(),
		"chainID":     existingEntry.GetChain(),
		"geolocation": strconv.FormatInt(int64(existingEntry.GetGeolocation()), 10),
		"moniker":     existingEntry.Description.Moniker,
		"description": existingEntry.Description.String(),
		"stake":       existingEntry.GetStake().Amount.String(),
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderUnstakeEventName, details, unstakeDescription)

	return nil
}

func (k Keeper) UnstakeEntryForce(ctx sdk.Context, chainID, provider, unstakeDescription string) error {
	providerAddr, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	existingEntry, entryExists := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, provider)
	if !entryExists {
		return utils.LavaFormatWarning("can't unstake Entry, stake entry not found for address", fmt.Errorf("stake entry not found"),
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "spec", Value: chainID},
		)
	}
	totalAmount := existingEntry.Stake.Amount
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, providerAddr)

	for _, delegation := range delegations {
		validator, found := k.stakingKeeper.GetValidator(ctx, delegation.GetValidatorAddr())
		if !found {
			continue
		}
		amount := validator.TokensFromShares(delegation.Shares).TruncateInt()
		if totalAmount.LT(amount) {
			amount = totalAmount
		}
		totalAmount = totalAmount.Sub(amount)
		err = k.dualstakingKeeper.UnbondFull(ctx, existingEntry.Vault, validator.OperatorAddress, existingEntry.Address, existingEntry.GetChain(), sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), amount), true)
		if err != nil {
			return utils.LavaFormatWarning("can't unbond self delegation", err,
				utils.LogAttr("provider", existingEntry.Address),
				utils.LogAttr("vault", existingEntry.Vault),
				utils.LogAttr("spec", chainID),
			)
		}

		if totalAmount.IsZero() {
			existingEntry, _ := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, provider)
			k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, chainID, existingEntry.Address)

			details := map[string]string{
				"address":     existingEntry.GetAddress(),
				"chainID":     existingEntry.GetChain(),
				"geolocation": strconv.FormatInt(int64(existingEntry.GetGeolocation()), 10),
				"description": existingEntry.Description.String(),
				"moniker":     existingEntry.Description.Moniker,
				"stake":       existingEntry.GetStake().Amount.String(),
			}
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProviderUnstakeEventName, details, unstakeDescription)

			return nil
		}
	}

	return nil
}

func (k Keeper) SlashDelegator(ctx sdk.Context, slashingInfo types.DelegatorSlashing) error {
	delAddr, err := sdk.AccAddressFromBech32(slashingInfo.Delegator)
	if err != nil {
		return err
	}
	total := slashingInfo.SlashingAmount.Amount

	// this method goes over all unbondings and tries to slash them
	slashUnbonding := func() {
		unbondings := k.stakingKeeper.GetUnbondingDelegations(ctx, delAddr, math.MaxUint16)

		for _, unbonding := range unbondings {
			totalBalance := sdk.ZeroInt()
			for _, entry := range unbonding.Entries {
				totalBalance = totalBalance.Add(entry.Balance)
			}
			if totalBalance.IsZero() {
				continue
			}

			slashingFactor := total.ToLegacyDec().QuoInt(totalBalance)
			slashingFactor = sdk.MinDec(sdk.OneDec(), slashingFactor)
			slashedAmount := k.stakingKeeper.SlashUnbondingDelegation(ctx, unbonding, 1, slashingFactor)
			slashedAmount = sdk.MinInt(total, slashedAmount)
			total = total.Sub(slashedAmount)

			if !total.IsPositive() {
				return
			}
		}
	}

	// try slashing existing unbondings
	slashUnbonding()
	if !total.IsPositive() {
		return nil
	}

	// we need to unbond from the delegator by force the amount left
	delegations := k.stakingKeeper.GetAllDelegatorDelegations(ctx, delAddr)
	tokensToUnbond := total
	for _, delegation := range delegations {
		validator, found := k.stakingKeeper.GetValidator(ctx, delegation.GetValidatorAddr())
		if !found {
			continue
		}
		amount := validator.TokensFromShares(delegation.Shares).TruncateInt()
		amount = sdk.MinInt(tokensToUnbond, amount)
		shares, err := validator.SharesFromTokensTruncated(amount)
		if err != nil {
			return utils.LavaFormatWarning("failed to get delegators shares", err,
				utils.Attribute{Key: "address", Value: delAddr},
				utils.Attribute{Key: "validator", Value: validator.OperatorAddress},
			)
		}

		_, err = k.stakingKeeper.Undelegate(ctx, delAddr, validator.GetOperator(), shares)
		if err != nil {
			return utils.LavaFormatWarning("can't unbond self delegation for slashing", err,
				utils.Attribute{Key: "address", Value: delAddr},
			)
		}

		tokensToUnbond = tokensToUnbond.Sub(amount)
		if !tokensToUnbond.IsPositive() {
			break
		}
	}

	if tokensToUnbond.IsPositive() {
		utils.LavaFormatWarning("delegator does not have enough delegations to slash", err,
			utils.Attribute{Key: "address", Value: delAddr},
		)
	}

	// try slashing existing unbondings
	slashUnbonding()

	return nil
}
