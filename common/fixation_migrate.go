package common

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

func prefixForErrors(from uint64) string {
	return fmt.Sprintf("FixationStore: migration from version %d", from)
}

var fixationMigrators = map[int]func(sdk.Context, *FixationStore) error{
	1: fixationMigrate1to2,
}

func (fs *FixationStore) MigrateVersion(ctx sdk.Context) (err error) {
	from := fs.getVersion(ctx)
	to := types.FixationVersion()

	for from < to {
		function, ok := fixationMigrators[int(from)]
		if !ok {
			return fmt.Errorf("%s not available", prefixForErrors(from))
		}

		err = function(ctx, fs)
		if err != nil {
			return err
		}

		from += 1
	}

	fs.setVersion(ctx, to)
	return nil
}
