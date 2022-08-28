package keeper

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper    types.BankKeeper
		accountKeeper types.AccountKeeper
		specKeeper    types.SpecKeeper

		fixationRegistries map[string]func(sdk.Context) any
		buffer             *bytes.Buffer
		enc                *gob.Encoder
		dec                *gob.Decoder
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper, specKeeper types.SpecKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	dec := gob.NewDecoder(&buffer)

	keeper := &Keeper{

		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,
		bankKeeper: bankKeeper, accountKeeper: accountKeeper, specKeeper: specKeeper,

		fixationRegistries: make(map[string]func(sdk.Context) any),
		buffer:             &buffer,
		enc:                enc,
		dec:                dec,
	}

	keeper.AddFixationRegistry(string(types.KeyEpochBlocks), func(ctx sdk.Context) any { return Keeper.EpochBlocksRaw(*keeper, ctx) })
	keeper.AddFixationRegistry(string(types.KeyEpochsToSave), func(ctx sdk.Context) any { return Keeper.EpochsToSaveRaw(*keeper, ctx) })
	keeper.AddFixationRegistry(string(types.KeyUnstakeHoldBlocks), func(ctx sdk.Context) any { return Keeper.UnstakeHoldBlocksRaw(*keeper, ctx) })

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any) {

	if _, ok := k.fixationRegistries[fixationKey]; ok {
		panic(fmt.Sprintf("duplicate fixation registry %s", fixationKey))
	}
	k.fixationRegistries[fixationKey] = getParamFunction
}

func (k *Keeper) GetFixationRegistries() map[string]func(sdk.Context) any {
	return k.fixationRegistries
}
