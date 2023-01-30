package lavaprotocol

import (
	"testing"

	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func createNewProviderHashesConsensus(f *FinalizationConsensus) ProviderHashesConsensus {
	return f.newProviderHashesConsensus(1, "testAddress", 111, map[int64]string{1: "test"}, &types.RelayReply{}, &types.RelayRequest{})
}

// Test the basic functionality of the finalization consensus
func TestEpochChange(t *testing.T) {
	finalization_consensus := &FinalizationConsensus{}
	finalization_consensus.NewEpoch(50)
	createNewProviderHashesConsensus(finalization_consensus)
	finalization_consensus.currentProviderHashesConsensus = append(finalization_consensus.currentProviderHashesConsensus,
		createNewProviderHashesConsensus(finalization_consensus))
	require.NotEmpty(t, finalization_consensus.currentProviderHashesConsensus)
	require.Equal(t, len(finalization_consensus.currentProviderHashesConsensus), 1)
	finalization_consensus.NewEpoch(100)
	require.Equal(t, finalization_consensus.currentEpoch, 100)
	require.Empty(t, finalization_consensus.currentProviderHashesConsensus)
	require.Equal(t, len(finalization_consensus.prevEpochProviderHashesConsensus), 1)
}
