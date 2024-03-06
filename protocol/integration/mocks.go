package integration_test

import (
	"context"
	"fmt"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/statetracker/updaters"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type mockConsumerStateTracker struct {
}

func (m *mockConsumerStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {

}

func (m *mockConsumerStateTracker) RegisterConsumerSessionManagerForPairingUpdates(ctx context.Context, consumerSessionManager *lavasession.ConsumerSessionManager) {

}
func (m *mockConsumerStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	return nil
}

func (m *mockConsumerStateTracker) RegisterFinalizationConsensusForUpdates(context.Context, *lavaprotocol.FinalizationConsensus) {

}

func (m *mockConsumerStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	return nil
}

func (m *mockConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, sameProviderConflict *conflicttypes.FinalizationConflict, conflictHandler common.ConflictHandlerInterface) error {
	return nil
}

func (m *mockConsumerStateTracker) GetConsumerPolicy(ctx context.Context, consumerAddress, chainID string) (*plantypes.Policy, error) {
	return &plantypes.Policy{
		ChainPolicies:         []plantypes.ChainPolicy{},
		GeolocationProfile:    1,
		TotalCuLimit:          10000,
		EpochCuLimit:          1000,
		MaxProvidersToPair:    5,
		SelectedProvidersMode: 0,
		SelectedProviders:     []string{},
	}, nil
}

func (m *mockConsumerStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return nil, fmt.Errorf("banana")
}
func (m *mockConsumerStateTracker) GetLatestVirtualEpoch() uint64 {
	return 0
}
