package common

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (tstore *TimerStore) prefixForErrors(from uint64) string {
	return fmt.Sprintf("TimerStore: migration from version %d", from)
}

var timerMigrators = map[int]func(sdk.Context, *TimerStore) error{
	// fill with map entrys like "1: timerMigrate1to2"
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
