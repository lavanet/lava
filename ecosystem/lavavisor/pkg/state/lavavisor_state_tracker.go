package lvstatetracker

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/statetracker"
	"github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
	protocoltypes "github.com/lavanet/lava/v4/x/protocol/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

// Lava visor doesn't require complicated state tracker, it just needs to periodically fetch the protocol version.
type LavaVisorStateTracker struct {
	stateQuery       *updaters.StateQuery
	averageBlockTime time.Duration
	ticker           *time.Ticker
	versionUpdater   *LavaVisorVersionUpdater
}

func NewLavaVisorStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (lvst *LavaVisorStateTracker, err error) {
	// validate chainId
	stateQuery := updaters.NewStateQuery(ctx, updaters.NewStateQueryAccessInst(clientCtx))
	status, err := stateQuery.Status(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("[Lavavisor] failed getting status", err)
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("[Lavavisor] Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}
	specQueryClient := spectypes.NewQueryClient(clientCtx)
	specResponse, err := statetracker.GetLavaSpecWithRetry(ctx, specQueryClient)
	if err != nil {
		utils.LavaFormatFatal("chain is missing Lava spec, cant initialize lavavisor", err)
	}
	lst := &LavaVisorStateTracker{stateQuery: stateQuery, averageBlockTime: time.Duration(specResponse.Spec.AverageBlockTime) * time.Millisecond}
	return lst, nil
}

func (lst *LavaVisorStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
	lst.versionUpdater = &LavaVisorVersionUpdater{VersionUpdater: updaters.VersionUpdater{
		VersionStateQuery:    lst.stateQuery,
		LastKnownVersion:     &updaters.ProtocolVersionResponse{Version: version, BlockNumber: "uninitialized"},
		VersionValidationInf: versionValidator,
	}}
	lst.ticker = time.NewTicker(lst.averageBlockTime)
	lst.versionUpdater.Update()
	go func() {
		for {
			select {
			case <-lst.ticker.C:
				lst.versionUpdater.Update()
				lst.ticker = time.NewTicker(lst.averageBlockTime)
			case <-ctx.Done():
				lst.ticker.Stop()
				return
			}
		}
	}()
}

func (lst *LavaVisorStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return lst.stateQuery.GetProtocolVersion(ctx)
}

type LavaVisorVersionUpdater struct {
	updaters.VersionUpdater
}

// monitor protocol version on each new block, this method overloads the update
// method of version updater in protocol because here we fetch the protocol version every block
// instead of listening to events which is not necessary in lava visor
func (vu *LavaVisorVersionUpdater) Update() {
	vu.Lock.Lock()
	defer vu.Lock.Unlock()
	// fetch updated version from consensus
	version, err := vu.VersionStateQuery.GetProtocolVersion(context.Background())
	if err != nil {
		utils.LavaFormatError("[Lavavisor] could not get version from node, its possible the node is down", err)
		return
	}
	vu.LastKnownVersion = version
	err = vu.ValidateProtocolVersion(vu.LastKnownVersion)
	if err != nil {
		utils.LavaFormatError("[Lavavisor] Validate Protocol Version Error", err)
	}
}
