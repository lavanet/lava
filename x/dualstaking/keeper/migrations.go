package keeper

import (
	"encoding/json"
	"fmt"

	_ "embed"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	dualstakingv4 "github.com/lavanet/lava/v2/x/dualstaking/migrations/v4"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
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
			for _, entry := range storage.StakeEntries {
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
				m.keeper.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, entry)
				err = m.keeper.DelegateFull(ctx, entry.Address, highestVal.OperatorAddress, entry.Address, chainID, stake)
				if err != nil {
					return err
				}
			}
		}
	}

	moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(types.ModuleName), m.keeper.stakingKeeper.BondDenom(ctx))
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
	moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(dualstakingtypes.BondedPoolName), m.keeper.stakingKeeper.BondDenom(ctx))
	if !moduleBalance.IsZero() {
		err := m.keeper.bankKeeper.BurnCoins(ctx, dualstakingtypes.BondedPoolName, sdk.NewCoins(moduleBalance))
		if err != nil {
			return err
		}
	}

	if !moduleBalance.IsZero() {
		moduleBalance = m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(dualstakingtypes.NotBondedPoolName), m.keeper.stakingKeeper.BondDenom(ctx))
		err := m.keeper.bankKeeper.BurnCoins(ctx, dualstakingtypes.NotBondedPoolName, sdk.NewCoins(moduleBalance))
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

		if d.Delegator == d.Provider || d.Provider == commontypes.EMPTY_PROVIDER {
			continue
		}

		originalAmount := d.Amount

		// zero the delegation amount in the fixation store
		d.Amount = sdk.NewCoin(m.keeper.stakingKeeper.BondDenom(ctx), sdk.ZeroInt())
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

		entry, found := m.keeper.epochstorageKeeper.GetStakeEntryByAddressCurrent(ctx, d.ChainID, d.Provider)
		if !found {
			continue
		}
		entry.DelegateTotal = entry.DelegateTotal.Sub(originalAmount)
		m.keeper.epochstorageKeeper.ModifyStakeEntryCurrent(ctx, d.ChainID, entry)

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
		diff, _, err := m.keeper.VerifyDelegatorBalance(ctx, d)
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

// MigrateVersion2To3 implements store migration: Create a self delegation for all providers, Make providers-validators delegations balance
func (m Migrator) MigrateVersion2To3(ctx sdk.Context) error {
	nextEpoch := m.keeper.epochstorageKeeper.GetCurrentNextEpoch(ctx)
	chains := m.keeper.specKeeper.GetAllChainIDs(ctx)
	OldStakeStorages := GetStakeStorages(ctx.ChainID())
	for _, chainID := range chains {
		storage, found := m.keeper.epochstorageKeeper.GetStakeStorageCurrent(ctx, chainID)
		if found {
			providers := map[string]interface{}{}
			stakeEntries := []epochstoragetypes.StakeEntry{}

			duplicated := 0
			missing := 0
			// remove duplicates
			for _, entry := range storage.StakeEntries {
				if _, ok := providers[entry.Address]; !ok {
					d, found := m.keeper.GetDelegation(ctx, entry.Address, entry.Address, entry.Chain, nextEpoch)
					if found {
						delegations, err := m.keeper.GetProviderDelegators(ctx, entry.Address, nextEpoch)
						if err == nil {
							entry.DelegateTotal.Amount = sdk.ZeroInt()
							for _, d := range delegations {
								if d.Delegator != d.Provider && d.ChainID == chainID {
									entry.DelegateTotal.Amount = entry.DelegateTotal.Amount.Add(d.Amount.Amount)
								}
							}
						} else {
							fmt.Println("didnt find delegations for:", entry.Address)
						}
						entry.Stake = d.Amount
						stakeEntries = append(stakeEntries, entry)
					} else {
						fmt.Println("didnt find self delegation for:", entry.Address)
					}

					providers[entry.Address] = struct{}{}
				} else {
					duplicated++
				}
			}

			// add back the old ones if they were deleted
			if len(OldStakeStorages) > 0 {
				deletedEntriesToAdd := OldStakeStorages[chainID]
				for _, entry := range deletedEntriesToAdd.StakeEntries {
					if _, ok := providers[entry.Address]; !ok {
						d, found := m.keeper.GetDelegation(ctx, entry.Address, entry.Address, entry.Chain, nextEpoch)
						if found {
							missing++
							delegations, err := m.keeper.GetProviderDelegators(ctx, entry.Address, nextEpoch)
							if err == nil {
								entry.DelegateTotal.Amount = sdk.ZeroInt()
								for _, d := range delegations {
									if d.Delegator != d.Provider && d.ChainID == chainID {
										entry.DelegateTotal.Amount = entry.DelegateTotal.Amount.Add(d.Amount.Amount)
									}
								}
							}
							entry.Stake = d.Amount
							stakeEntries = append(stakeEntries, entry)
						}
					}
				}
			}

			utils.LavaFormatInfo("Migrator for chain id providers", utils.LogAttr("chainID", chainID),
				utils.LogAttr("duplicated", duplicated),
				utils.LogAttr("missing", missing))
			storage.StakeEntries = stakeEntries
			m.keeper.epochstorageKeeper.SetStakeStorageCurrent(ctx, storage.Index, storage)
		}
	}
	return nil
}

var (
	good_stakeStorage_lava_staging_4 []byte
	good_stakeStorage_lava_testnet_2 []byte
)

func GetStakeStorages(lavaChainID string) map[string]epochstoragetypes.StakeStorage {
	var payload []byte
	if lavaChainID == "lava-testnet-2" {
		payload = good_stakeStorage_lava_testnet_2
	} else if lavaChainID == "lava-staging-4" {
		payload = good_stakeStorage_lava_staging_4
	}

	var stakestorages []epochstoragetypes.StakeStorage
	err := json.Unmarshal(payload, &stakestorages)
	if err != nil {
		utils.LavaFormatWarning("Could not unmarshal stakestorages", err, utils.LogAttr("chainID", lavaChainID), utils.LogAttr("PayloadLen", len(payload)))
	}

	stakestoragesMap := map[string]epochstoragetypes.StakeStorage{}
	for _, stakestorage := range stakestorages {
		stakestoragesMap[stakestorage.Index] = stakestorage
	}
	return stakestoragesMap
}

// MigrateVersion3To4 sets the DisableDualstakingHook flag to false
func (m Migrator) MigrateVersion3To4(ctx sdk.Context) error {
	m.keeper.SetDisableDualstakingHook(ctx, false)
	return nil
}

// MigrateVersion4To5 change the DelegatorReward object amount's field from sdk.Coin to sdk.Coins
func (m Migrator) MigrateVersion4To5(ctx sdk.Context) error {
	delegatorRewards := m.GetAllDelegatorRewardV4(ctx)
	for _, dr := range delegatorRewards {
		delegatorReward := dualstakingtypes.DelegatorReward{
			Delegator: dr.Delegator,
			Provider:  dr.Provider,
			ChainId:   dr.ChainId,
			Amount:    sdk.NewCoins(dr.Amount),
		}
		m.keeper.RemoveDelegatorReward(ctx, dualstakingtypes.DelegationKey(dr.Provider, dr.Delegator, dr.ChainId))
		m.keeper.SetDelegatorReward(ctx, delegatorReward)
	}

	params := dualstakingtypes.DefaultParams()
	m.keeper.SetParams(ctx, params)
	return nil
}

func (m Migrator) GetAllDelegatorRewardV4(ctx sdk.Context) (list []dualstakingv4.DelegatorRewardv4) {
	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(dualstakingtypes.DelegatorRewardKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val dualstakingv4.DelegatorRewardv4
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
