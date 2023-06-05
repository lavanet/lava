package pairing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the uniquePaymentStorageClientProvider
	for _, elem := range genState.UniquePaymentStorageClientProviderList {
		k.SetUniquePaymentStorageClientProvider(ctx, elem)
	}
	// Set all the providerPaymentStorage
	for _, elem := range genState.ProviderPaymentStorageList {
		k.SetProviderPaymentStorage(ctx, elem)
	}
	// Set all the epochPayments
	for _, elem := range genState.EpochPaymentsList {
		k.SetEpochPayments(ctx, elem)
	}
	// Set all the badgeUsedCu
	for _, elem := range genState.BadgeUsedCuList {
		k.SetBadgeUsedCu(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.UniquePaymentStorageClientProviderList = k.GetAllUniquePaymentStorageClientProvider(ctx)
	genesis.ProviderPaymentStorageList = k.GetAllProviderPaymentStorage(ctx)
	genesis.EpochPaymentsList = k.GetAllEpochPayments(ctx)
	genesis.BadgeUsedCuList = k.GetAllBadgeUsedCu(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
