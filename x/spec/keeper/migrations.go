package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/spec/types"
	typesv1 "github.com/lavanet/lava/v4/x/spec/types/migrations/v1"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	store := prefix.NewStore(ctx.KVStore(m.keeper.storeKey), types.KeyPrefix(types.SpecKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var specV1 typesv1.Spec
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &specV1)

		var spec types.Spec
		m.keeper.cdc.MustUnmarshal(iterator.Value(), &spec)

		for acIndex, apiCollection := range specV1.ApiCollections {
			for eIndex, extension := range apiCollection.Extensions {
				spec.ApiCollections[acIndex].Extensions[eIndex].CuMultiplier = uint64(extension.CuMultiplier)
			}
		}

		m.keeper.SetSpec(ctx, spec)
	}

	return nil
}
