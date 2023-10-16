package statetracker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	blockInFirstEpoch  = uint64(1)
	blockInSecondEpoch = uint64(2)
)

type epochStateQueryTest struct {
	init bool
}

func (esqt *epochStateQueryTest) CurrentEpochStart(ctx context.Context) (uint64, error) {
	if !esqt.init {
		esqt.init = true
		return blockInFirstEpoch, nil
	}
	return blockInSecondEpoch, nil
}

type epochUpdatable struct {
	updatedBlock uint64
}

func (esqt *epochUpdatable) UpdateEpoch(epoch uint64) {
	esqt.updatedBlock = epoch
}

func TestEpochUpdaterDelay(t *testing.T) {
	epochUpdater := NewEpochUpdater(&epochStateQueryTest{})
	updatable := &epochUpdatable{updatedBlock: 0}
	epochUpdater.RegisterEpochUpdatable(context.Background(), updatable, 3)
	require.Equal(t, blockInFirstEpoch, updatable.updatedBlock)
	epochUpdater.Update(2)
	require.Equal(t, blockInFirstEpoch, updatable.updatedBlock)
	epochUpdater.Update(3)
	require.Equal(t, blockInFirstEpoch, updatable.updatedBlock)
	epochUpdater.Update(4)
	require.Equal(t, blockInFirstEpoch, updatable.updatedBlock)

	// after 3 blocks delay we should now get an update
	epochUpdater.Update(5)
	require.Equal(t, blockInSecondEpoch, updatable.updatedBlock)
	epochUpdater.Update(6)
	require.Equal(t, blockInSecondEpoch, updatable.updatedBlock)
}
