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
	// Set all the clientPaymentStorage
	for _, elem := range genState.ClientPaymentStorageList {
		k.SetClientPaymentStorage(ctx, elem)
	}
	// Set all the epochPayments
	for _, elem := range genState.EpochPaymentsList {
		k.SetEpochPayments(ctx, elem)
	}
	// Set all the fixatedServicersToPair
	for _, elem := range genState.FixatedServicersToPairList {
		k.SetFixatedServicersToPair(ctx, elem)
	}
	// Set all the fixatedStakeToMaxCu
	for _, elem := range genState.FixatedStakeToMaxCuList {
		k.SetFixatedStakeToMaxCu(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	k.PushFixatedServicersToPair(ctx, 0, 0)
	k.PushFixatedStakeToMaxCu(ctx, 0, 0)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.UniquePaymentStorageClientProviderList = k.GetAllUniquePaymentStorageClientProvider(ctx)
	genesis.ClientPaymentStorageList = k.GetAllClientPaymentStorage(ctx)
	genesis.EpochPaymentsList = k.GetAllEpochPayments(ctx)
	genesis.FixatedServicersToPairList = k.GetAllFixatedServicersToPair(ctx)
	genesis.FixatedStakeToMaxCuList = k.GetAllFixatedStakeToMaxCu(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
