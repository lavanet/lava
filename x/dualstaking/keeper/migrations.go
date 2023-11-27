package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// ConvertProviderStakeToSelfDelegation does:
// 1. zero out each provider stake and return their money
// 2. use that same money to make the providers self delegate
func (m Migrator) ConvertProviderStakeToSelfDelegation(ctx sdk.Context) error {
	// get highest staked validator
	validatorsByPower := m.keeper.stakingKeeper.GetBondedValidatorsByPower(ctx)
	highestVal := validatorsByPower[0]

	// loop over all providers
	chains := m.keeper.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range chains {
		storage, found := m.keeper.epochstorageKeeper.GetStakeStorageCurrent(ctx, chainID)
		if found {
			for i, entry := range storage.StakeEntries {
				// return the providers all their coins
				addr, err := sdk.AccAddressFromBech32(entry.Address)
				if err != nil {
					return err
				}

				err = m.keeper.bankKeeper.MintCoins(ctx, types.ModuleName, []sdk.Coin{entry.Stake})
				if err != nil {
					return err
				}

				err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{entry.Stake})
				if err != nil {
					utils.LavaFormatError("failed to send coins from module to account", err,
						utils.Attribute{Key: "Account", Value: addr},
					)
				}

				// create self delegation, this will increase the stake entry, we need to fix that by reseting the stake before delegating
				stake := entry.Stake
				entry.Stake.Amount = sdk.ZeroInt()
				m.keeper.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, entry, uint64(i))
				err = m.keeper.DelegateFull(ctx, entry.Address, highestVal.OperatorAddress, entry.Address, chainID, stake)
				if err != nil {
					return err
				}
			}
		}
	}

	moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(types.ModuleName), epochstoragetypes.TokenDenom)
	if !moduleBalance.IsZero() {
		err := m.keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(moduleBalance))
		if err != nil {
			return err
		}
	}

	return nil
}

// HandleProviderDelegators does:
// 1. merge the deprecated bonded and not-bonded pool funds
// 2. return the providers' delegators money back from the merged pool
// 3. use the same money to delegate to both the original delegation's provider and highest staked validator
func (m Migrator) HandleProviderDelegators(ctx sdk.Context) error {
	delegationsInds := m.keeper.delegationFS.GetAllEntryIndices(ctx)
	nextEpoch := m.keeper.epochstorageKeeper.GetCurrentNextEpoch(ctx)

	// burn all coins from the pools
	moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(dualstakingtypes.BondedPoolName), epochstoragetypes.TokenDenom)
	if !moduleBalance.IsZero() {
		err := m.keeper.bankKeeper.BurnCoins(ctx, dualstakingtypes.BondedPoolName, sdk.NewCoins(moduleBalance))
		if err != nil {
			return err
		}
	}

	if !moduleBalance.IsZero() {
		moduleBalance = m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(dualstakingtypes.NotBondedPoolName), epochstoragetypes.TokenDenom)
		err = m.keeper.bankKeeper.BurnCoins(ctx, dualstakingtypes.NotBondedPoolName, sdk.NewCoins(moduleBalance))
		if err != nil {
			return err
		}
	}

	// give money back to delegators from the bonded pool
	originalDelegations := []dualstakingtypes.Delegation{}
	for _, ind := range delegationsInds {
		// find the delegation and keep its original form
		var d dualstakingtypes.Delegation
		block, _, _, found := m.keeper.delegationFS.FindEntryDetailed(ctx, ind, nextEpoch, &d)
		if !found {
			continue
		}

		if d.Delegator == d.Provider || d.Provider == EMPTY_PROVIDER {
			continue
		}

		providerAddr, err := sdk.AccAddressFromBech32(d.Provider)
		if err != nil {
			return err
		}

		originalAmount := d.Amount

		// zero the delegation amount in the fixation store
		d.Amount = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())
		m.keeper.delegationFS.ModifyEntry(ctx, ind, block, &d)

		// give money back from the bonded pool
		delegatorAddr, err := sdk.AccAddressFromBech32(d.Delegator)
		if err != nil {
			return err
		}

		err = m.keeper.bankKeeper.MintCoins(ctx, types.ModuleName, []sdk.Coin{originalAmount})
		if err != nil {
			return err
		}

		err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, delegatorAddr, sdk.Coins{originalAmount})
		if err != nil {
			return err
		}

		entry, found, index := m.keeper.epochstorageKeeper.GetStakeEntryByAddressCurrent(ctx, d.ChainID, providerAddr)
		if !found {
			continue
		}
		entry.DelegateTotal = entry.DelegateTotal.Sub(originalAmount)
		m.keeper.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, d.ChainID, entry, index)

		originalDelegations = append(originalDelegations, d)
	}

	// get highest staked validator and delegate to it
	validatorsByPower := m.keeper.stakingKeeper.GetBondedValidatorsByPower(ctx)
	highestVal := validatorsByPower[0]
	for _, d := range originalDelegations {
		err := m.keeper.DelegateFull(ctx, d.Delegator, highestVal.OperatorAddress, d.Provider, d.ChainID, d.Amount)
		if err != nil {
			return err
		}
	}

	return nil
}

// HandleValidatorsDelegators does:
// 1. get each validator's delegators
// 2. delegate the amount of their delegation to the empty provider (using the AfterDelegationModified hook)
func (m Migrator) HandleValidatorsDelegators(ctx sdk.Context) error {
	// get all validators
	validators := m.keeper.stakingKeeper.GetAllValidators(ctx)

	// for each validator+delegator, run the AfterDelegationModified to delegate to empty provider
	for _, v := range validators {
		delegations := m.keeper.stakingKeeper.GetValidatorDelegations(ctx, v.GetOperator())
		for _, d := range delegations {
			err := m.keeper.Hooks().AfterDelegationModified(ctx, d.GetDelegatorAddr(), v.GetOperator())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// VerifyDelegationsBalance gets all delegators (from both providers and validators)
// and verifies that each delegator has the same amount of delegations in both the provider
// and validator sides
func (m Migrator) VerifyDelegationsBalance(ctx sdk.Context) error {
	delegationsInds := m.keeper.delegationFS.GetAllEntryIndices(ctx)
	validators := m.keeper.stakingKeeper.GetAllValidators(ctx)
	nextEpoch := m.keeper.epochstorageKeeper.GetCurrentNextEpoch(ctx)

	// get all the providers' delegators
	var delegators []sdk.AccAddress
	for _, ind := range delegationsInds {
		var d dualstakingtypes.Delegation
		found := m.keeper.delegationFS.FindEntry(ctx, ind, nextEpoch, &d)
		if !found {
			continue
		}
		delegatorAddr, err := sdk.AccAddressFromBech32(d.Delegator)
		if err != nil {
			return err
		}
		delegators = append(delegators, delegatorAddr)
	}

	// get all the validators' delegators
	for _, v := range validators {
		delegations := m.keeper.stakingKeeper.GetValidatorDelegations(ctx, v.GetOperator())
		for _, d := range delegations {
			delegatorAddr, err := sdk.AccAddressFromBech32(d.DelegatorAddress)
			if err != nil {
				return err
			}
			delegators = append(delegators, delegatorAddr)
		}
	}

	// verify delegations balance for each delegator
	for _, d := range delegators {
		diff, err := m.keeper.VerifyDelegatorBalance(ctx, d)
		if err != nil {
			return err
		}

		if !diff.IsZero() {
			return utils.LavaFormatError("delegations balance migration failed", fmt.Errorf("delegator not balanced"),
				utils.Attribute{Key: "delegator", Value: d.String()},
				utils.Attribute{Key: "diff", Value: diff.String()},
			)
		}
	}

	return nil
}

// MigrateVersion1To2 implements store migration: Create a self delegation for all providers, Make providers-validators delegations balance
func (m Migrator) MigrateVersion1To2(ctx sdk.Context) error {
	// first balance all validators
	err := m.HandleValidatorsDelegators(ctx)
	if err != nil {
		return err
	}

	// create providers self delegations
	err = m.ConvertProviderStakeToSelfDelegation(ctx)
	if err != nil {
		return err
	}

	// convert the rest of the delegations
	err = m.HandleProviderDelegators(ctx)
	if err != nil {
		return err
	}

	// verify the balance once again to make sure
	err = m.VerifyDelegationsBalance(ctx)
	if err != nil {
		return err
	}

	return nil
}
