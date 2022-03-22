package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	usertypes "github.com/lavanet/lava/x/user/types"
)

type EvidenceKeeper interface {
	// Methods imported from evidence should be defined here
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) types.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx sdk.Context, moduleName string, amounts sdk.Coins) error
	// Methods imported from bank should be defined here
}

type SpecKeeper interface {
	IsSpecFoundAndActive(ctx sdk.Context, specName string) (bool, bool, uint64)
	IsSpecIDFoundAndActive(ctx sdk.Context, id uint64) (bool, bool)
	GetSpec(ctx sdk.Context, id uint64) (val spectypes.Spec, found bool)
}

type UserKeeper interface {
	GetSpecStakeStorage(sdk.Context, string) (usertypes.SpecStakeStorage, bool)
}
