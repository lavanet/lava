package pairing

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the uniquePaymentStorageClientProvider
	for _, elem := range genState.UniqueEpochSessions {
		provider, project, chainID, sessionID, err := types.DecodeUniqueEpochSessionKey(elem.UniqueEpochSession)
		if err != nil {
			utils.LavaFormatError("could not decode UniqueEpochSessionKey", err, utils.LogAttr("key", elem))
			continue
		}
		k.SetUniqueEpochSession(ctx, elem.Epoch, provider, project, chainID, sessionID)
	}
	// Set all the providerPaymentStorage
	for _, elem := range genState.ProviderEpochCus {
		k.SetProviderEpochCu(ctx, elem.Epoch, elem.Provider, elem.ChainId, elem.ProviderEpochCu)
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

	epochs, uniqueEpochSessions := k.GetAllUniqueEpochSessionStore(ctx)
	for i := range epochs {
		ues := types.UniqueEpochSessionGenesis{
			Epoch:              epochs[i],
			UniqueEpochSession: uniqueEpochSessions[i],
		}
		genesis.UniqueEpochSessions = append(genesis.UniqueEpochSessions, ues)
	}

	epochs, keys, providerEpochCus := k.GetAllProviderEpochCuStore(ctx)
	for i := range epochs {
		provider, chainID, err := types.DecodeProviderEpochCuKey(keys[i])
		if err != nil {
			utils.LavaFormatError("could not decode ProviderEpochCu key", err, utils.LogAttr("key", keys[i]))
			continue
		}
		pec := types.ProviderEpochCuGenesis{
			Epoch:           epochs[i],
			Provider:        provider,
			ChainId:         chainID,
			ProviderEpochCu: providerEpochCus[i],
		}
		genesis.ProviderEpochCus = append(genesis.ProviderEpochCus, pec)
	}

	epochs, keys, providerConsumerEpochCus := k.GetAllProviderConsumerEpochCuStore(ctx)
	for i := range epochs {
		provider, project, chainID, err := types.DecodeProviderConsumerEpochCuKey(keys[i])
		if err != nil {
			utils.LavaFormatError("could not decode ProviderConsumerEpochCu key", err, utils.LogAttr("key", keys[i]))
			continue
		}
		pcec := types.ProviderConsumerEpochCuGenesis{
			Epoch:                   epochs[i],
			Provider:                provider,
			Project:                 project,
			ChainId:                 chainID,
			ProviderConsumerEpochCu: providerConsumerEpochCus[i],
		}
		genesis.ProviderConsumerEpochCus = append(genesis.ProviderConsumerEpochCus, pcec)
	}

	genesis.BadgeUsedCuList = k.GetAllBadgeUsedCu(ctx)
	genesis.BadgesTS = k.ExportBadgesTimers(ctx)
	genesis.ProviderQosFS = k.ExportProviderQoS(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
