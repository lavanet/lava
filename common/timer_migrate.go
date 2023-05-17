package common

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

func (tstore *TimerStore) prefixForErrors(from uint64) string {
	return fmt.Sprintf("TimerStore: migration from version %d", from)
}

var timerMigrators = map[int]func(sdk.Context, *TimerStore) error{
	1: timerMigrate1to2,
}

func (tstore *TimerStore) MigrateVersion(ctx sdk.Context) (err error) {
	from := tstore.getVersion(ctx)
	to := TimerVersion()

	for from < to {
		function, ok := timerMigrators[int(from)]
		if !ok {
			return fmt.Errorf("%s not available", prefixForErrors(from))
		}

		err = function(ctx, tstore)
		if err != nil {
			return err
		}

		from += 1
	}

	tstore.setVersion(ctx, to)
	return nil
}

// timerMigrate1to2: fix refcounts
//   - convert entry keys from "<block>" to "<block>index (otherwise timer for an entry
//     with some block will be overwritten by timer for another entry with same block.
func timerMigrate1to2(ctx sdk.Context, tstore *TimerStore) error {
	for _, which := range []types.TimerType{types.BlockHeight, types.BlockTime} {
		store := tstore.getStoreTimer(ctx, which)

		iterator := sdk.KVStorePrefixIterator(store, []byte{})
		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			value, key := types.DecodeBlockAndKey(iterator.Key())
			key_v1 := types.EncodeKey(value)
			key_v2 := types.EncodeBlockAndKey(value, key)
			store.Set(key_v2, iterator.Value())
			store.Delete(key_v1)
		}
	}

	return nil
}
