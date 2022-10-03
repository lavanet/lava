package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/conflict/keeper"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/stretchr/testify/require"
	tmdb "github.com/tendermint/tm-db"
)

func ConflictKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	epochstorage, ctx := EpochstorageKeeper(t)
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := ctx.MultiStore().(storetypes.CommitMultiStore)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"ConflictParams",
	)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		nil,
		nil,
		nil,
		epochstorage,
		nil,
	)

	ctx = ctx.WithMultiStore(stateStore)
	epochstorage.EpochBlocksRaw(ctx)
	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
