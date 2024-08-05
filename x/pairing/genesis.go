package pairing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the uniquePaymentStorageClientProvider
	for _, elem := range genState.UniqueEpochSessions {
		k.SetUniqueEpochSession(ctx, elem.Epoch, elem.Provider, elem.Project, elem.ChainId, elem.SessionId)
	}
	// Set all the providerPaymentStorage
	for _, elem := range genState.ProviderEpochCus {
		k.SetProviderEpochCu(ctx, elem.Epoch, elem.Provider, elem.ChainId, elem.ProviderEpochCu)
	}
	// Set all the ProviderEpochComplainedCus
	for _, elem := range genState.ProviderEpochComplainedCus {
		k.SetProviderEpochComplainerCu(ctx, elem.Epoch, elem.Provider, elem.ChainId, elem.ProviderEpochComplainerCu)
	}
	// Set all the epochPayments
	for _, elem := range genState.ProviderConsumerEpochCus {
		k.SetProviderConsumerEpochCu(ctx, elem.Epoch, elem.Provider, elem.Project, elem.ChainId, elem.ProviderConsumerEpochCu)
	}
	// Set all the badgeUsedCu
	for _, elem := range genState.BadgeUsedCuList {
		k.SetBadgeUsedCu(ctx, elem)
	}

	k.InitBadgeTimers(ctx, genState.BadgesTS)
	k.InitProviderQoS(ctx, genState.ProviderQosFS)
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.UniqueEpochSessions = k.GetAllUniqueEpochSessionStore(ctx)
	genesis.ProviderEpochCus = k.GetAllProviderEpochCuStore(ctx)
	genesis.ProviderEpochComplainedCus = k.GetAllProviderEpochComplainerCuStore(ctx)
	genesis.ProviderConsumerEpochCus = k.GetAllProviderConsumerEpochCuStore(ctx)
	genesis.BadgeUsedCuList = k.GetAllBadgeUsedCu(ctx)
	genesis.BadgesTS = k.ExportBadgesTimers(ctx)
	genesis.ProviderQosFS = k.ExportProviderQoS(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
