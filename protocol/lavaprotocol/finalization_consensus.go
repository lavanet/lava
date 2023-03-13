package lavaprotocol

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"golang.org/x/exp/slices"
)

type FinalizationConsensus struct {
	currentProviderHashesConsensus   []ProviderHashesConsensus
	prevEpochProviderHashesConsensus []ProviderHashesConsensus
	providerDataContainersMu         sync.RWMutex
	currentEpoch                     uint64
}

type ProviderHashesConsensus struct {
	FinalizedBlocksHashes map[int64]string
	agreeingProviders     map[string]providerDataContainer
}

type providerDataContainer struct {
	LatestFinalizedBlock  int64
	LatestBlockTime       time.Time
	FinalizedBlocksHashes map[int64]string
	SigBlocks             []byte
	SessionId             uint64
	BlockHeight           int64
	RelayNum              uint64
	LatestBlock           int64
	// TODO:: keep relay request for conflict reporting
}

func GetLatestFinalizedBlock(latestBlock int64, blockDistanceForFinalizedData int64) int64 {
	finalization_criteria := blockDistanceForFinalizedData
	return latestBlock - finalization_criteria
}

func (fc *FinalizationConsensus) newProviderHashesConsensus(blockDistanceForFinalizedData int64, providerAcc string, latestBlock int64, finalizedBlocks map[int64]string, reply *pairingtypes.RelayReply, req *pairingtypes.RelaySession) ProviderHashesConsensus {
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:  GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData),
		LatestBlockTime:       time.Now(),
		FinalizedBlocksHashes: finalizedBlocks,
		SigBlocks:             reply.SigBlocks,
		SessionId:             req.SessionId,
		RelayNum:              req.RelayNum,
		BlockHeight:           req.BlockHeight,
		LatestBlock:           latestBlock,
	}
	providerDataContainers := map[string]providerDataContainer{}
	providerDataContainers[providerAcc] = newProviderDataContainer
	return ProviderHashesConsensus{
		FinalizedBlocksHashes: finalizedBlocks,
		agreeingProviders:     providerDataContainers,
	}
}

func (fc *FinalizationConsensus) insertProviderToConsensus(blockDistanceForFinalizedData int64, consensus *ProviderHashesConsensus, finalizedBlocks map[int64]string, latestBlock int64, reply *pairingtypes.RelayReply, req *pairingtypes.RelaySession, providerAcc string) {
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:  GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData),
		LatestBlockTime:       time.Now(),
		FinalizedBlocksHashes: finalizedBlocks,
		SigBlocks:             reply.SigBlocks,
		SessionId:             req.SessionId,
		RelayNum:              req.RelayNum,
		BlockHeight:           req.BlockHeight,
		LatestBlock:           latestBlock,
	}
	consensus.agreeingProviders[providerAcc] = newProviderDataContainer

	for blockNum, blockHash := range finalizedBlocks {
		consensus.FinalizedBlocksHashes[blockNum] = blockHash
	}
}

// Compare finalized block hashes with previous providers
// Looks for discrepancy with current epoch providers
// if no conflicts, insert into consensus and break
// create new consensus group if no consensus matched
// check for discrepancy with old epoch
// checks if there is a consensus mismatch between hashes provided by different providers
func (fc *FinalizationConsensus) UpdateFinalizedHashes(blockDistanceForFinalizedData int64, providerAddress string, latestBlock int64, finalizedBlocks map[int64]string, req *pairingtypes.RelaySession, reply *pairingtypes.RelayReply) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	fc.providerDataContainersMu.Lock()
	defer fc.providerDataContainersMu.Unlock()

	if len(fc.currentProviderHashesConsensus) == 0 && len(fc.prevEpochProviderHashesConsensus) == 0 {
		newHashConsensus := fc.newProviderHashesConsensus(blockDistanceForFinalizedData, providerAddress, latestBlock, finalizedBlocks, reply, req)
		fc.currentProviderHashesConsensus = append(make([]ProviderHashesConsensus, 0), newHashConsensus)
	} else {
		// Looks for discrepancy with current epoch providers
		for _, consensus := range fc.currentProviderHashesConsensus {
			err := fc.discrepancyChecker(finalizedBlocks, consensus)
			if err != nil {
				// TODO: bring the other data as proof
				finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
				// TODO: before returning, we need to create a new ProviderHashesConsensus group if there isn't one that matches
				// TODO: check there is no matching consensus group and add him to a new one
				// create new consensus group if no consensus matched
				// newHashConsensus := fc.newProviderHashesConsensus(blockDistanceForFinalizedData, providerAddress, latestBlock, finalizedBlocks, reply, req)
				// fc.currentProviderHashesConsensus = append(make([]ProviderHashesConsensus, 0), newHashConsensus)
				return finalizationConflict, utils.LavaFormatError("Simulation: Conflict found in discrepancyChecker", err, nil)
			}

			// if no discrepency with this group -> insert into consensus
			fc.insertProviderToConsensus(blockDistanceForFinalizedData, &consensus, finalizedBlocks, latestBlock, reply, req, providerAddress)
			// keep comparing with other groups, if there is a new message with a conflict we need to report it too
		}

		// check for discrepancy with old epoch
		for idx, consensus := range fc.prevEpochProviderHashesConsensus {
			err := fc.discrepancyChecker(finalizedBlocks, consensus)
			if err != nil {
				// TODO: bring the other data as proof
				finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: reply}
				return finalizationConflict, utils.LavaFormatError("Simulation: prev epoch Conflict found in discrepancyChecker", err, &map[string]string{"Consensus idx": strconv.Itoa(idx), "provider": providerAddress})
			}
		}
	}

	return finalizationConflict, nil
}

func (fc *FinalizationConsensus) discrepancyChecker(finalizedBlocksA map[int64]string, consensus ProviderHashesConsensus) (errRet error) {
	var toIterate map[int64]string   // the smaller map between the two to compare
	var otherBlocks map[int64]string // the other map

	if len(finalizedBlocksA) < len(consensus.FinalizedBlocksHashes) {
		toIterate = finalizedBlocksA
		otherBlocks = consensus.FinalizedBlocksHashes
	} else {
		toIterate = consensus.FinalizedBlocksHashes
		otherBlocks = finalizedBlocksA
	}
	// Iterate over smaller array, looks for mismatching hashes between the inputs
	for blockNum, blockHash := range toIterate {
		if otherHash, ok := otherBlocks[blockNum]; ok {
			if blockHash != otherHash {
				// TODO: gather discrepancy data
				return utils.LavaFormatError("Simulation: reliability discrepancy, different hashes detected for block", HashesConsunsusError, &map[string]string{"blockNum": strconv.FormatInt(blockNum, 10), "Hashes": fmt.Sprintf("%s vs %s", blockHash, otherHash), "toIterate": fmt.Sprintf("%v", toIterate), "otherBlocks": fmt.Sprintf("%v", otherBlocks)})
			}
		}
	}

	return nil
}

func (fc *FinalizationConsensus) NewEpoch(epoch uint64) {
	fc.providerDataContainersMu.Lock()
	defer fc.providerDataContainersMu.Unlock()

	if fc.currentEpoch < epoch {
		// means it's time to refresh the epoch
		fc.prevEpochProviderHashesConsensus = fc.currentProviderHashesConsensus
		fc.currentProviderHashesConsensus = []ProviderHashesConsensus{}
		fc.currentEpoch = epoch
	}
}

// returns the expected latest block, does the calculation on finalized entries then extrapolates the ending based on blockDistance
func (s *FinalizationConsensus) ExpectedBlockHeight(chainParser chainlib.ChainParser) (expectedBlockHeight int64, numOfProviders int) {
	s.providerDataContainersMu.RLock()
	defer s.providerDataContainersMu.RUnlock()
	allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData, _ := chainParser.ChainBlockStats()
	averageBlockTime_ms := averageBlockTime
	listExpectedBlockHeights := []int64{}

	var highestBlockNumber int64 = 0
	FindHighestBlockNumber := func(listProviderHashesConsensus []ProviderHashesConsensus) int64 {
		for _, providerHashesConsensus := range listProviderHashesConsensus {
			for _, providerDataContainer := range providerHashesConsensus.agreeingProviders {
				if highestBlockNumber < providerDataContainer.LatestFinalizedBlock {
					highestBlockNumber = providerDataContainer.LatestFinalizedBlock
				}
			}
		}
		return highestBlockNumber
	}
	highestBlockNumber = FindHighestBlockNumber(s.prevEpochProviderHashesConsensus) // update the highest in place
	highestBlockNumber = FindHighestBlockNumber(s.currentProviderHashesConsensus)

	now := time.Now()
	calcExpectedBlocks := func(listProviderHashesConsensus []ProviderHashesConsensus) []int64 {
		listExpectedBH := []int64{}
		for _, providerHashesConsensus := range listProviderHashesConsensus {
			for _, providerDataContainer := range providerHashesConsensus.agreeingProviders {
				expected := providerDataContainer.LatestFinalizedBlock + (now.Sub(providerDataContainer.LatestBlockTime) / averageBlockTime_ms).Milliseconds() // interpolation
				// limit the interpolation to the highest seen block height
				if expected > highestBlockNumber {
					expected = highestBlockNumber
				}
				listExpectedBH = append(listExpectedBH, expected)
			}
		}
		return listExpectedBH
	}
	listExpectedBlockHeights = append(listExpectedBlockHeights, calcExpectedBlocks(s.prevEpochProviderHashesConsensus)...)
	listExpectedBlockHeights = append(listExpectedBlockHeights, calcExpectedBlocks(s.currentProviderHashesConsensus)...)

	median := func(data []int64) int64 {
		slices.Sort(data)

		var median int64
		data_len := len(data)
		if data_len == 0 {
			return 0
		} else if data_len%2 == 0 {
			median = (data[data_len/2-1] + data[data_len/2]/2.0)
		} else {
			median = data[data_len/2]
		}
		return median
	}
	return median(listExpectedBlockHeights) - allowedBlockLagForQosSync + int64(blockDistanceForFinalizedData), len(listExpectedBlockHeights)
}
