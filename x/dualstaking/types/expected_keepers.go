package types

import (
	context "context"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	fixationstoretypes "github.com/lavanet/lava/v4/x/fixationstore/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderPool, recipientPool string, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type EpochstorageKeeper interface {
	GetStakeEntryCurrent(ctx sdk.Context, chainID string, address string) (value epochstoragetypes.StakeEntry, found bool)
	SetStakeEntryCurrent(ctx sdk.Context, stakeEntry epochstoragetypes.StakeEntry)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64)
	RemoveStakeEntryCurrent(ctx sdk.Context, chainID string, address string)
	GetStakeEntry(ctx sdk.Context, epoch uint64, chainID string, provider string) (val epochstoragetypes.StakeEntry, found bool)
	GetMetadata(ctx sdk.Context, provider string) (epochstoragetypes.ProviderMetadata, error)
	SetMetadata(ctx sdk.Context, metadata epochstoragetypes.ProviderMetadata)
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
	ValidatorByConsAddr(context.Context, sdk.ConsAddress) (stakingtypes.ValidatorI, error)
	UnbondingTime(ctx context.Context) (time.Duration, error)
	GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]stakingtypes.Delegation, error)
	GetDelegatorValidator(ctx context.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetValidatorDelegations(ctx context.Context, valAddr sdk.ValAddress) ([]stakingtypes.Delegation, error)
	BondDenom(ctx context.Context) (string, error)
	ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (shares math.LegacyDec, err error)
	Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec) (time.Time, math.Int, error)
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (newShares math.LegacyDec, err error)
	GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstoretypes.FixationStore
}
