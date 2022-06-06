package keeper

import (
	"context"
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

type Keepers struct {
	Epochstorage  epochstoragekeeper.Keeper
	Spec          speckeeper.Keeper
	Pairing       pairingkeeper.Keeper
	BankKeeper    mockBankKeeper
	AccountKeeper mockAccountKeeper
}

type servers struct {
	EpochServer   epochtypes.MsgServer
	SpecServer    spectypes.MsgServer
	PairingServer types.MsgServer
}

func InitAllKeepers(t testing.TB) (*servers, *Keepers, context.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	pairingStoreKey := sdk.NewKVStoreKey(types.StoreKey)
	pairingMemStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)
	stateStore.MountStoreWithDB(pairingStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(pairingMemStoreKey, sdk.StoreTypeMemory, nil)

	specStoreKey := sdk.NewKVStoreKey(spectypes.StoreKey)
	specMemStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)
	stateStore.MountStoreWithDB(specStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(specMemStoreKey, sdk.StoreTypeMemory, nil)

	epochStoreKey := sdk.NewKVStoreKey(epochtypes.StoreKey)
	epochMemStoreKey := storetypes.NewMemoryStoreKey(epochtypes.MemStoreKey)
	stateStore.MountStoreWithDB(epochStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	epochparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		epochStoreKey,
		epochMemStoreKey,
		"EpochstorageParams",
	)

	pairingparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		pairingStoreKey,
		pairingMemStoreKey,
		"PairingParams",
	)

	specparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		specStoreKey,
		specMemStoreKey,
		"SpecParams",
	)

	ks := Keepers{}
	ks.AccountKeeper = mockAccountKeeper{}
	ks.BankKeeper = mockBankKeeper{balance: make(map[string]sdk.Coins), moduleBank: make(map[string]map[string]sdk.Coins)}
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, ks.Epochstorage)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	ks.Pairing.SetParams(ctx, types.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochtypes.DefaultParams())

	ss := servers{}
	ss.EpochServer = epochstoragekeeper.NewMsgServerImpl(ks.Epochstorage)
	ss.SpecServer = speckeeper.NewMsgServerImpl(ks.Spec)
	ss.PairingServer = pairingkeeper.NewMsgServerImpl(ks.Pairing)

	return &ss, &ks, sdk.WrapSDKContext(ctx)
}
