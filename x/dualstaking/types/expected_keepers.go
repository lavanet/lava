package types

import (
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationstoretypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	BurnCoins(ctx sdk.Context, name string, amt sdk.Coins) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderPool, recipientPool string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type EpochstorageKeeper interface {
	GetStakeEntryByAddressCurrent(ctx sdk.Context, chainID string, address string) (value epochstoragetypes.StakeEntry, found bool)
	ModifyStakeEntryCurrent(ctx sdk.Context, chainID string, stakeEntry epochstoragetypes.StakeEntry)
	UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64)
	UnstakeHoldBlocksStatic(ctx sdk.Context, block uint64) (res uint64)
	GetStakeEntryForProviderEpoch(ctx sdk.Context, chainID string, selectedProvider string, epoch uint64) (entry epochstoragetypes.StakeEntry, found bool)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64)
	GetStakeStorageCurrent(ctx sdk.Context, chainID string) (epochstoragetypes.StakeStorage, bool)
	SetStakeStorageCurrent(ctx sdk.Context, chainID string, stakeStorage epochstoragetypes.StakeStorage)
	RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, address string) error
	AppendUnstakeEntry(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry, unstakeHoldBlocks uint64)
	GetUnstakeHoldBlocks(ctx sdk.Context, chainID string) uint64
	UnstakeEntryByAddress(ctx sdk.Context, address string) (value epochstoragetypes.StakeEntry, found bool)
	// Methods imported from epochstorage should be defined here
}

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (active bool, found bool, providerType spectypes.Spec_ProvidersTypes)
	GetContributorReward(ctx sdk.Context, chainId string) (contributors []sdk.AccAddress, percentage math.LegacyDec)
	GetSpec(ctx sdk.Context, index string) (val spectypes.Spec, found bool)
	GetAllChainIDs(ctx sdk.Context) (chainIDs []string)
	GetMinStake(ctx sdk.Context, chainID string) sdk.Coin
}

type StakingKeeper interface {
	ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI
	UnbondingTime(ctx sdk.Context) time.Duration
	GetAllDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress) []stakingtypes.Delegation
	GetDelegatorValidator(ctx sdk.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	GetDelegation(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, found bool)
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	GetValidatorDelegations(ctx sdk.Context, valAddr sdk.ValAddress) (delegations []stakingtypes.Delegation)
	BondDenom(ctx sdk.Context) string
	ValidateUnbondAmount(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (shares sdk.Dec, err error)
	Undelegate(ctx sdk.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdk.Dec) (time.Time, error)
	Delegate(ctx sdk.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (newShares sdk.Dec, err error)
	GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator
	GetAllValidators(ctx sdk.Context) (validators []stakingtypes.Validator)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstoretypes.FixationStore
}
