package lvstatetracker

import (
	"context"
	"fmt"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/utils"

	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type EventTracker struct {
	lock               sync.RWMutex
	clientCtx          client.Context
	blockResults       *ctypes.ResultBlockResults
	latestUpdatedBlock int64
}

func (et *EventTracker) updateBlockResults(latestBlock int64) (err error) {
	ctx := context.Background()
	var blockResults *ctypes.ResultBlockResults
	fmt.Println("latestBlock: ", latestBlock)
	fmt.Println("et.ClientCtx: ", et.clientCtx)

	if latestBlock == 0 {
		res, err := et.clientCtx.Client.Status(ctx)
		fmt.Println("res: ", res)
		fmt.Println("err: ", err)

		if err != nil {
			return utils.LavaFormatWarning("could not get latest block height and requested latestBlock = 0", err)
		}
		latestBlock = res.SyncInfo.LatestBlockHeight
	}
	blockResults, err = et.clientCtx.Client.BlockResults(ctx, &latestBlock)
	if err != nil {
		return err
	}
	// lock for update after successful block result query
	et.lock.Lock()
	defer et.lock.Unlock()
	et.latestUpdatedBlock = latestBlock
	et.blockResults = blockResults
	return nil
}

func (et *EventTracker) getLatestVersionEvents() (updated bool) {
	et.lock.RLock()
	defer et.lock.RUnlock()
	for _, event := range et.blockResults.EndBlockEvents {
		if event.Type == utils.EventPrefix+"param_change" {
			for _, attribute := range event.Attributes {
				if string(attribute.Key) == "param" && string(attribute.Value) == "Version" {
					return true
				}
			}
		}
	}
	return false
}
