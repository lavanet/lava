package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type SpecKeeper interface {
	// Methods imported from spec should be defined here
	GetAllChainIDs(ctx sdk.Context) (chainIDs []string)
	IsSpecFoundAndActive(ctx sdk.Context, chainID string) (foundAndActive, found bool, providersType spectypes.Spec_ProvidersTypes)
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
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	// Methods imported from bank should be defined here
}

type StakingKeeper interface {
	BondDenom(ctx sdk.Context) string
}
