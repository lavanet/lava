package finalizationconsensus

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/utils/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

type testPlays struct {
	name                   string
	finalizationInsertions []finalizationTestInsertion
	consensusHashesCount   int
}
type finalizationTestInsertion struct {
	providerAddr    string
	latestBlock     uint64
	finalizedBlocks map[int64]string
	success         bool
	relaySession    *pairingtypes.RelaySession
	relayReply      *pairingtypes.RelayReply
}

func createStubHashes(from, to uint64, identifier string) map[int64]string {
	ret := map[int64]string{}
	for i := from; i <= to; i++ {
		ret[int64(i)] = strconv.Itoa(int(i)) + identifier
	}
	return ret
}

func finalizationInsertionForProviders(chainID string, epoch, latestBlock uint64, startProvider, providersNum int, success bool, identifier string, blocksInFinalizationProof, blockDistanceForFinalizedData uint32) (rets []finalizationTestInsertion) {
	fromBlock := latestBlock - uint64(blockDistanceForFinalizedData)
	for i := startProvider; i < startProvider+providersNum; i++ {
		rets = append(rets, finalizationTestInsertion{
			providerAddr:    "lava@provider" + strconv.Itoa(i),
			latestBlock:     latestBlock,
			finalizedBlocks: createStubHashes(fromBlock, fromBlock+uint64(blocksInFinalizationProof)-1, identifier),
			success:         success,
			relaySession: &pairingtypes.RelaySession{
				SpecId:                chainID,
				ContentHash:           []byte{},
				SessionId:             uint64(i),
				CuSum:                 0,
				Provider:              "lava@provider" + strconv.Itoa(i),
				RelayNum:              1,
				QosReport:             &pairingtypes.QualityOfServiceReport{},
				Epoch:                 int64(epoch),
				UnresponsiveProviders: nil,
				LavaChainId:           "lava",
				Sig:                   []byte{},
			},
			relayReply: &pairingtypes.RelayReply{
				LatestBlock:           int64(latestBlock),
				FinalizedBlocksHashes: []byte{},
				SigBlocks:             []byte{},
				Metadata:              []pairingtypes.Metadata{},
			},
		})
	}
	return rets
}

func TestConsensusHashesInsertion(t *testing.T) {
	chainsToTest := []string{"APT1", "LAV1", "ETH1"}
	for _, chainID := range chainsToTest {
		ctx := context.Background()
		chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, chainID, "0", func(http.ResponseWriter, *http.Request) {}, nil, "../../../", nil)
		if closeServer != nil {
			defer closeServer()
		}
		require.NoError(t, err)
		require.NotNil(t, chainParser)
		epoch := uint64(200)

		_, _, blockDistanceForFinalizedData, blocksInFinalizationProof := chainParser.ChainBlockStats()
		require.Greater(t, blocksInFinalizationProof, uint32(0))

		shouldSucceedOnOneBeofreOrAfter := blocksInFinalizationProof <= 1

		consensusHashesCountOnOneBeofreOrAfter := 2
		if shouldSucceedOnOneBeofreOrAfter {
			consensusHashesCountOnOneBeofreOrAfter = 1
		}
		playbook := []testPlays{
			{
				name:                 "happy-flow",
				consensusHashesCount: 1,
				finalizationInsertions: append(append(finalizationInsertionForProviders(chainID, epoch, 100, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 101, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
					finalizationInsertionForProviders(chainID, epoch, 102, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "happy-flow-with-gap",
				consensusHashesCount: 1,
				finalizationInsertions: append(finalizationInsertionForProviders(chainID, epoch, 100, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 100+uint64(blocksInFinalizationProof), 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "mismatch-with-self",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(chainID, epoch, 100, 0, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 100, 0, 1, false, "A", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "mismatch-with-others",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(chainID, epoch, 100, 1, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 100, 0, 1, false, "A", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "mismatch-with-others-one-after",
				consensusHashesCount: consensusHashesCountOnOneBeofreOrAfter,
				finalizationInsertions: append(finalizationInsertionForProviders(chainID, epoch, 100, 1, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 101, 0, 1, shouldSucceedOnOneBeofreOrAfter, "A", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "mismatch-with-others-one-before",
				consensusHashesCount: consensusHashesCountOnOneBeofreOrAfter,
				finalizationInsertions: append(finalizationInsertionForProviders(chainID, epoch, 100, 1, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 99, 0, 1, shouldSucceedOnOneBeofreOrAfter, "A", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
			{
				name:                 "mismatch-three-groups",
				consensusHashesCount: 3,
				finalizationInsertions: append(append(finalizationInsertionForProviders(chainID, epoch, 100, 0, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
					finalizationInsertionForProviders(chainID, epoch, 100, 1, 1, false, "A", blocksInFinalizationProof, blockDistanceForFinalizedData)...), finalizationInsertionForProviders(chainID, epoch, 100, 2, 1, false, "B", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
			},
		}
		for _, play := range playbook {
			t.Run(chainID+":"+play.name, func(t *testing.T) {
				finalizationConsensus := &FinalizationConsensus{}
				finalizationConsensus.NewEpoch(epoch)
				// check updating hashes works
				for _, insertion := range play.finalizationInsertions {
					_, err := finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), insertion.providerAddr, insertion.finalizedBlocks, insertion.relaySession, insertion.relayReply)
					if insertion.success {
						require.NoError(t, err, "failed insertion when was supposed to succeed, provider %s, latest block %d", insertion.providerAddr, insertion.latestBlock)
					} else {
						require.Error(t, err)
					}
				}
				require.Len(t, finalizationConsensus.currentProviderHashesConsensus, play.consensusHashesCount)
			})
		}
	}
}

func TestQoS(t *testing.T) {
	decToSet, _ := sdk.NewDecFromStr("0.05") // test values fit 0.05 Availability requirements
	lavasession.AvailabilityPercentage = decToSet
	rand.InitRandomSeed()
	chainsToTest := []string{"APT1", "LAV1", "ETH1"}
	for _, chainID := range chainsToTest {
		t.Run(chainID, func(t *testing.T) {
			ctx := context.Background()
			chainParser, _, _, closeServer, _, err := chainlib.CreateChainLibMocks(ctx, chainID, "0", func(http.ResponseWriter, *http.Request) {}, nil, "../../../", nil)
			if closeServer != nil {
				defer closeServer()
			}
			require.NoError(t, err)
			require.NotNil(t, chainParser)
			epoch := uint64(200)

			consumerSessionsWithProvider := lavasession.ConsumerSessionsWithProvider{
				PublicLavaAddress: "",
				Endpoints:         []*lavasession.Endpoint{},
				Sessions:          map[int64]*lavasession.SingleConsumerSession{},
				MaxComputeUnits:   10000,
				UsedComputeUnits:  0,
				PairingEpoch:      epoch,
			}
			singleConsumerSession, _, err := consumerSessionsWithProvider.GetConsumerSessionInstanceFromEndpoint(&lavasession.EndpointConnection{}, 1)
			require.NoError(t, err)
			require.NotNil(t, singleConsumerSession)

			allowedBlockLagForQosSync, _, blockDistanceForFinalizedData, blocksInFinalizationProof := chainParser.ChainBlockStats()
			require.Greater(t, blocksInFinalizationProof, uint32(0))

			finalizationInsertions := append(append(finalizationInsertionForProviders(chainID, epoch, 200, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
				finalizationInsertionForProviders(chainID, epoch, 201, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
				finalizationInsertionForProviders(chainID, epoch, 202, 0, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...)

			newEpoch := epoch + 20
			finalizationInsertionsAfterEpoch := append(append(finalizationInsertionForProviders(chainID, newEpoch, 203, 2, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData),
				finalizationInsertionForProviders(chainID, epoch, 204, 2, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...),
				finalizationInsertionForProviders(chainID, epoch, 205, 2, 3, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)...)

			finalizationConsensus := &FinalizationConsensus{}
			finalizationConsensus.NewEpoch(epoch)
			for _, insertion := range finalizationInsertions {
				_, err := finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), insertion.providerAddr, insertion.finalizedBlocks, insertion.relaySession, insertion.relayReply)
				require.NoError(t, err, "failed insertion when was supposed to succeed, provider %s, latest block %d", insertion.providerAddr, insertion.latestBlock)
			}
			plannedExpectedBH := int64(202) // this is the most advanced in all finalizations
			require.Len(t, finalizationConsensus.currentProviderHashesConsensus, 1)
			expectedBH, numOfProviders := finalizationConsensus.ExpectedBlockHeight(chainParser)
			require.Equal(t, uint64(plannedExpectedBH), finalizationConsensus.LatestBlock())
			require.Equal(t, 3, numOfProviders)
			require.Equal(t, plannedExpectedBH-allowedBlockLagForQosSync, expectedBH)

			// now advance an epoch to make it interesting
			finalizationConsensus.NewEpoch(newEpoch)
			for _, insertion := range finalizationInsertionsAfterEpoch {
				_, err := finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), insertion.providerAddr, insertion.finalizedBlocks, insertion.relaySession, insertion.relayReply)
				require.NoError(t, err, "failed insertion when was supposed to succeed, provider %s, latest block %d", insertion.providerAddr, insertion.latestBlock)
			}
			plannedExpectedBH = 205 // this is the most advanced in all finalizations after epoch change
			expectedBH, numOfProviders = finalizationConsensus.ExpectedBlockHeight(chainParser)
			require.Equal(t, 5, numOfProviders)
			require.Equal(t, plannedExpectedBH-allowedBlockLagForQosSync, expectedBH)

			currentLatency := time.Millisecond
			expectedLatency := time.Millisecond
			latestServicedBlock := expectedBH
			singleConsumerSession.CalculateQoS(currentLatency, expectedLatency, expectedBH-latestServicedBlock, numOfProviders, 1)
			require.Equal(t, uint64(1), singleConsumerSession.QoSInfo.AnsweredRelays)
			require.Equal(t, uint64(1), singleConsumerSession.QoSInfo.TotalRelays)
			require.Equal(t, int64(1), singleConsumerSession.QoSInfo.SyncScoreSum)
			require.Equal(t, int64(1), singleConsumerSession.QoSInfo.TotalSyncScore)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Availability)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Sync)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Latency)

			latestServicedBlock = expectedBH + 1
			singleConsumerSession.CalculateQoS(currentLatency, expectedLatency, expectedBH-latestServicedBlock, numOfProviders, 1)
			require.Equal(t, uint64(2), singleConsumerSession.QoSInfo.AnsweredRelays)
			require.Equal(t, uint64(2), singleConsumerSession.QoSInfo.TotalRelays)
			require.Equal(t, int64(2), singleConsumerSession.QoSInfo.SyncScoreSum)
			require.Equal(t, int64(2), singleConsumerSession.QoSInfo.TotalSyncScore)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Availability)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Sync)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Latency)

			singleConsumerSession.QoSInfo.TotalRelays++ // this is how we add a failure
			singleConsumerSession.CalculateQoS(currentLatency, expectedLatency, expectedBH-latestServicedBlock, numOfProviders, 1)
			require.Equal(t, uint64(3), singleConsumerSession.QoSInfo.AnsweredRelays)
			require.Equal(t, uint64(4), singleConsumerSession.QoSInfo.TotalRelays)
			require.Equal(t, int64(3), singleConsumerSession.QoSInfo.SyncScoreSum)
			require.Equal(t, int64(3), singleConsumerSession.QoSInfo.TotalSyncScore)

			require.Equal(t, sdk.ZeroDec(), singleConsumerSession.QoSInfo.LastQoSReport.Availability) // because availability below 95% is 0
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Sync)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Latency)

			latestServicedBlock = expectedBH - 1 // is one block below threshold
			singleConsumerSession.CalculateQoS(currentLatency, expectedLatency*2, expectedBH-latestServicedBlock, numOfProviders, 1)
			require.Equal(t, uint64(4), singleConsumerSession.QoSInfo.AnsweredRelays)
			require.Equal(t, uint64(5), singleConsumerSession.QoSInfo.TotalRelays)
			require.Equal(t, int64(3), singleConsumerSession.QoSInfo.SyncScoreSum)
			require.Equal(t, int64(4), singleConsumerSession.QoSInfo.TotalSyncScore)

			require.Equal(t, sdk.ZeroDec(), singleConsumerSession.QoSInfo.LastQoSReport.Availability) // because availability below 95% is 0
			require.Equal(t, sdk.MustNewDecFromStr("0.75"), singleConsumerSession.QoSInfo.LastQoSReport.Sync)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Latency)
			latestServicedBlock = expectedBH + 1
			// add in a loop so availability goes above 95%
			for i := 5; i < 100; i++ {
				singleConsumerSession.CalculateQoS(currentLatency, expectedLatency*2, expectedBH-latestServicedBlock, numOfProviders, 1)
			}
			require.Equal(t, sdk.MustNewDecFromStr("0.8"), singleConsumerSession.QoSInfo.LastQoSReport.Availability) // because availability below 95% is 0
			require.Equal(t, sdk.MustNewDecFromStr("0.989898989898989898"), singleConsumerSession.QoSInfo.LastQoSReport.Sync)
			require.Equal(t, sdk.OneDec(), singleConsumerSession.QoSInfo.LastQoSReport.Latency)

			finalizationInsertionsSpreadBlocks := []finalizationTestInsertion{
				finalizationInsertionForProviders(chainID, epoch, 200, 0, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)[0],
				finalizationInsertionForProviders(chainID, epoch, 201, 1, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)[0],
				finalizationInsertionForProviders(chainID, epoch, 202, 2, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)[0],
				finalizationInsertionForProviders(chainID, epoch, 200, 3, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)[0],
				finalizationInsertionForProviders(chainID, epoch, 201, 4, 1, true, "", blocksInFinalizationProof, blockDistanceForFinalizedData)[0],
			}

			finalizationConsensus = &FinalizationConsensus{}
			finalizationConsensus.NewEpoch(epoch)
			for _, insertion := range finalizationInsertionsSpreadBlocks {
				_, err := finalizationConsensus.UpdateFinalizedHashes(int64(blockDistanceForFinalizedData), insertion.providerAddr, insertion.finalizedBlocks, insertion.relaySession, insertion.relayReply)
				require.NoError(t, err, "failed insertion when was supposed to succeed, provider %s, latest block %d", insertion.providerAddr, insertion.latestBlock)
			}

			plannedExpectedBH = int64(201) // this is the most advanced in all finalizations
			require.Len(t, finalizationConsensus.currentProviderHashesConsensus, 1)
			expectedBH, numOfProviders = finalizationConsensus.ExpectedBlockHeight(chainParser)
			require.Equal(t, 5, numOfProviders)
			require.Equal(t, plannedExpectedBH-allowedBlockLagForQosSync, expectedBH)

			now := time.Now()
			interpolation := InterpolateBlocks(now, now.Add(-2*time.Millisecond), time.Millisecond)
			require.Equal(t, int64(2), interpolation)
			interpolation = InterpolateBlocks(now, now.Add(-5*time.Millisecond), time.Millisecond)
			require.Equal(t, int64(5), interpolation)
			interpolation = InterpolateBlocks(now, now.Add(5*time.Millisecond), time.Millisecond)
			require.Equal(t, int64(0), interpolation)
		})
	}
}
