package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingkeeper "github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"

	speckeeper "github.com/lavanet/lava/x/spec/keeper"
)

type keepers struct {
	epochstorage  epochstoragekeeper.Keeper
	spec          speckeeper.Keeper
	pairing       pairingkeeper.Keeper
	bankKeeper    types.BankKeeper
	accountKeeper types.AccountKeeper
}

func InitAllKeepers(t testing.TB) (*keepers, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, sdk.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	epochparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"EpochstorageParams",
	)

	pairingparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"PairingParams",
	)

	specparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SpecParams",
	)

	ks := keepers{}
	ks.accountKeeper = &mockAccountKeeper{}
	ks.bankKeeper = &mockBankKeeper{}
	ks.spec = *speckeeper.NewKeeper(cdc, storeKey, memStoreKey, specparamsSubspace)
	ks.epochstorage = *epochstoragekeeper.NewKeeper(cdc, storeKey, memStoreKey, epochparamsSubspace, ks.bankKeeper, ks.accountKeeper, ks.spec)
	ks.pairing = *pairingkeeper.NewKeeper(cdc, storeKey, memStoreKey, pairingparamsSubspace, ks.bankKeeper, ks.accountKeeper, ks.spec, ks.epochstorage)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	ks.pairing.SetParams(ctx, types.DefaultParams())
	ks.spec.SetParams(ctx, spectypes.DefaultParams())
	ks.epochstorage.SetParams(ctx, epochtypes.DefaultParams())

	return &ks, ctx
}
