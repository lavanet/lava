package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

const EpochsToAddForFluctuations = 2

func (k Keeper) AddSecondaryAddressesToProvider(ctx sdk.Context, senderAddr sdk.AccAddress, secondaryAddresses []string) error {
	for _, granteeAddr := range secondaryAddresses {
		grantee, err := sdk.AccAddressFromBech32(granteeAddr)
		if err != nil {
			return utils.LavaFormatWarning("invalid secondary address", err,
				utils.Attribute{Key: "provider", Value: senderAddr.String()},
			)
		}
		allowance, err := types.NewProviderSecondaryAddressAllowance(k.AllowedSecondaryAddressMessageTypes())
		if err != nil {
			return utils.LavaFormatWarning("invalid grant allowance", err)
		}
		err = k.feeGrantKeeper.GrantAllowance(ctx, senderAddr, grantee, allowance)
		if err != nil {
			return utils.LavaFormatWarning("can't set secondary key allowance", err,
				utils.Attribute{Key: "granteeAddr", Value: granteeAddr}, utils.Attribute{Key: "secondaryAddressProvider", Value: k.GetProviderFromAddress(ctx, granteeAddr, ctx.BlockHeight())},
			)
		}
		err = k.SetProviderSecondaryAddress(ctx, senderAddr.String(), granteeAddr)
		if err != nil {
			return utils.LavaFormatWarning("can't set secondary key, it already exists in a provider", err,
				utils.Attribute{Key: "granteeAddr", Value: granteeAddr}, utils.Attribute{Key: "secondaryAddressProvider", Value: k.GetProviderFromAddress(ctx, granteeAddr, ctx.BlockHeight())},
			)
		}
	}
	return nil
}

func (k Keeper) GetProviderFromAddress(ctx sdk.Context, secondaryAddress string, height int64) string {
	return secondaryAddress
}

func (k Keeper) SetProviderSecondaryAddress(ctx sdk.Context, provider string, secondaryAddress string) error {
	return nil
}

func (k Keeper) RemoveProviderSecondaryAddress(ctx sdk.Context, provider string, secondaryAddress string) error {
	return nil
}

// revokes only after revocation period allowing the entries to be affected only after being deleted
func (k Keeper) RemoveSecondaryAddressesFromProvider(ctx sdk.Context, senderAddr sdk.AccAddress, secondaryAddresses []string) error {
	for _, granteeAddr := range secondaryAddresses {
		grantee, err := sdk.AccAddressFromBech32(granteeAddr)
		if err != nil {
			return utils.LavaFormatWarning("invalid secondary address", err,
				utils.Attribute{Key: "provider", Value: senderAddr.String()},
			)
		}
		epochsToSave, err := k.epochStorageKeeper.EpochsToSave(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			return err
		}
		downTimeParams := k.downtimeKeeper.GetParams(ctx)
		revocationTime := downTimeParams.EpochDuration * time.Duration(epochsToSave+EpochsToAddForFluctuations) // add a const to allow fluctuations in epoch time
		allowance := types.EmptyProviderSecondaryAddressAllowance(ctx, revocationTime)
		err = k.feeGrantKeeper.UpdateAllowance(ctx, senderAddr, grantee, allowance)
		if err != nil {
			return utils.LavaFormatWarning("can't remove secondary key allowance", err,
				utils.Attribute{Key: "granteeAddr", Value: granteeAddr}, utils.Attribute{Key: "secondaryAddressProvider", Value: k.GetProviderFromAddress(ctx, granteeAddr, ctx.BlockHeight())},
			)
		}
		err = k.RemoveProviderSecondaryAddress(ctx, senderAddr.String(), granteeAddr)
		if err != nil {
			return utils.LavaFormatWarning("can't remove secondary key", err,
				utils.Attribute{Key: "granteeAddr", Value: granteeAddr}, utils.Attribute{Key: "secondaryAddressProvider", Value: k.GetProviderFromAddress(ctx, granteeAddr, ctx.BlockHeight())},
			)
		}
	}
	return nil
}

func (k Keeper) SecondaryAddressesChange(ctx sdk.Context, senderAddr sdk.AccAddress, existingSecondaryAddresses []string, newSecondaryAddresses []string) error {
	existingSecondaryAddressesMap := map[string]bool{}
	// map existing secondary addresses
	for _, granteeAddr := range existingSecondaryAddresses {
		existingSecondaryAddressesMap[granteeAddr] = false
	}
	secondaryAddressesToAdd := []string{}
	for _, granteeAddr := range newSecondaryAddresses {
		if _, ok := existingSecondaryAddressesMap[granteeAddr]; !ok {
			secondaryAddressesToAdd = append(secondaryAddressesToAdd, granteeAddr)
			existingSecondaryAddressesMap[granteeAddr] = true
		}
	}

	secondaryAddressesToDelete := []string{}
	for _, granteeAddr := range existingSecondaryAddresses {
		if !existingSecondaryAddressesMap[granteeAddr] {
			// meaning it didn't exist in the new list
			secondaryAddressesToDelete = append(secondaryAddressesToDelete, granteeAddr)
		}
	}

	err := k.AddSecondaryAddressesToProvider(ctx, senderAddr, secondaryAddressesToAdd)
	if err != nil {
		return err
	}
	return k.RemoveSecondaryAddressesFromProvider(ctx, senderAddr, secondaryAddressesToDelete)
}
