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

// MigrateVersion1To2 implements store migration: Create a self delegation for all providers
func (m Migrator) MigrateVersion1To2(ctx sdk.Context) error {
	chains := m.keeper.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range chains {
		storage, found := m.keeper.epochstorageKeeper.GetStakeStorageCurrent(ctx, chainID)
		if found {
			for i, entry := range storage.StakeEntries {
				// first return the providers all their coins
				addr, err := sdk.AccAddressFromBech32(entry.Address)
				if err != nil {
					return err
				}
				fmt.Printf("entry.Address: %v\n", entry.Address)

				moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(types.ModuleName), epochstoragetypes.TokenDenom)
				if moduleBalance.IsLT(entry.Stake) {
					return fmt.Errorf("insufficient balance to unstake %s (current balance: %s)", entry.Stake, moduleBalance)
				}
				err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{entry.Stake})
				if err != nil {
					return fmt.Errorf("failed to send coins from module to %s: %w", addr, err)
				}

				// create self delegation, this will increase the stake entry, we need to fix that by reseting the stake before delegating
				stake := entry.Stake
				entry.Stake.Amount = sdk.ZeroInt()
				m.keeper.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, entry, uint64(i))

				err = m.keeper.Delegate(ctx, entry.Address, entry.Address, chainID, stake)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// MigrateVersion2To3 implements store migration: Make providers-validators delegations balance
func (m Migrator) MigrateVersion2To3(ctx sdk.Context) error {
	delegationsInds := m.keeper.delegationFS.GetAllEntryIndices(ctx)
	nextEpoch := m.keeper.epochstorageKeeper.GetCurrentNextEpoch(ctx)

	// move all funds from unbonded pool to bonded pool
	notBondedPoolAddr := m.keeper.accountKeeper.GetModuleAddress(dualstakingtypes.NotBondedPoolName)
	notBondedPoolAmount := m.keeper.bankKeeper.GetBalance(ctx, notBondedPoolAddr, epochstoragetypes.TokenDenom)
	if !notBondedPoolAmount.IsZero() {
		err := m.keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, dualstakingtypes.NotBondedPoolName, dualstakingtypes.BondedPoolName, sdk.Coins{notBondedPoolAmount})
		if err != nil {
			return err
		}
	}

	// get highest staked validator and delegate to it
	validatorsByPower := m.keeper.stakingKeeper.GetBondedValidatorsByPower(ctx)
	highestVal := validatorsByPower[0]

	list := []dualstakingtypes.Delegation{}
	for _, ind := range delegationsInds {
		var d dualstakingtypes.Delegation
		block, _, _, found := m.keeper.delegationFS.FindEntryDetailed(ctx, ind, nextEpoch, &d)
		if !found {
			continue
		}

		if d.Delegator == d.Provider {
			continue
		}
		list = append(list, d)
		fmt.Printf("d.Delegator: %v\n", d.Delegator)
		amountTest := d.Amount
		d.Amount = sdk.NewCoin("ulava", sdk.ZeroInt())
		m.keeper.delegationFS.ModifyEntry(ctx, ind, block, &d)

		delegatorAddr, err := sdk.AccAddressFromBech32(d.Delegator)
		if err != nil {
			return err
		}
		err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, dualstakingtypes.BondedPoolName, delegatorAddr, sdk.Coins{amountTest})
		if err != nil {
			return err
		}
	}

	for _, d := range list {
		err := m.keeper.DelegateFull(ctx, d.Delegator, highestVal.OperatorAddress, d.Provider, d.ChainID, d.Amount)
		if err != nil {
			return err
		}
	}

	// *** Handle validator delegators - stake to empty provider ***
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

			// verify the delegator is in balance
			diff, err := m.keeper.VerifyDelegatorBalance(ctx, d.GetDelegatorAddr())
			if err != nil {
				return err
			}

			if !diff.IsZero() {
				return utils.LavaFormatError("delegations balance migration failed", fmt.Errorf("delegator not balanced"),
					utils.Attribute{Key: "delegator", Value: d.DelegatorAddress},
					utils.Attribute{Key: "validator", Value: v.OperatorAddress},
					utils.Attribute{Key: "diff", Value: diff.String()},
				)
			}
		}
	}

	return nil
}
