package lavaprotocol

import (
	"context"
	"net/http"
	"strconv"
	"testing"

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

func TestConsensusHashesInsertion(t *testing.T) {
	chainsToTest := []string{"APT1", "LAV1", "ETH1"}
	for _, chainID := range chainsToTest {

		ctx := context.Background()
		chainParser, _, _, closeServer, err := chainlib.CreateChainLibMocks(ctx, chainID, "0", func(http.ResponseWriter, *http.Request) {}, "../../")
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
		singleConsumerSession, _, err := consumerSessionsWithProvider.GetConsumerSessionInstanceFromEndpoint(&lavasession.Endpoint{
			NetworkAddress:     "",
			Enabled:            true,
			Client:             nil,
			ConnectionRefusals: 0,
		}, 1)
		require.NoError(t, err)
		require.NotNil(t, singleConsumerSession)

		_, _, blockDistanceForFinalizedData, blocksInFinalizationProof := chainParser.ChainBlockStats()
		require.Greater(t, blocksInFinalizationProof, uint32(0))
		createStubHashes := func(from uint64, to uint64, identifier string) map[int64]string {
			ret := map[int64]string{}
			for i := from; i <= to; i++ {
				ret[int64(i)] = strconv.Itoa(int(i)) + identifier
			}
			return ret
		}

		finalizationInsertionForProviders := func(latestBlock uint64, startProvider int, providersNum int, success bool, identifier string) (rets []finalizationTestInsertion) {
			fromBlock := latestBlock - uint64(blockDistanceForFinalizedData)
			for i := startProvider; i < startProvider+providersNum; i++ {
				rets = append(rets, finalizationTestInsertion{
					providerAddr:    "lava@provider" + strconv.Itoa(i),
					latestBlock:     latestBlock,
					finalizedBlocks: createStubHashes(fromBlock, fromBlock+uint64(blocksInFinalizationProof), identifier),
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
						UnresponsiveProviders: []byte{},
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

		playbook := []testPlays{
			{
				name:                 "happy-flow",
				consensusHashesCount: 1,
				finalizationInsertions: append(append(finalizationInsertionForProviders(100, 0, 3, true, ""),
					finalizationInsertionForProviders(101, 0, 3, true, "")...),
					finalizationInsertionForProviders(102, 0, 3, true, "")...),
			},
			{
				name:                 "happy-flow-with-gap",
				consensusHashesCount: 1,
				finalizationInsertions: append(finalizationInsertionForProviders(100, 0, 3, true, ""),
					finalizationInsertionForProviders(100+uint64(blocksInFinalizationProof), 0, 3, true, "")...),
			},
			{
				name:                 "mismatch-with-self",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(100, 0, 1, true, ""),
					finalizationInsertionForProviders(100, 0, 1, false, "A")...),
			},
			{
				name:                 "mismatch-with-others",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(100, 1, 3, true, ""),
					finalizationInsertionForProviders(100, 0, 1, false, "A")...),
			},
			{
				name:                 "mismatch-with-others-one-after",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(100, 1, 3, true, ""),
					finalizationInsertionForProviders(101, 0, 1, false, "A")...),
			},
			{
				name:                 "mismatch-with-others-one-before",
				consensusHashesCount: 2,
				finalizationInsertions: append(finalizationInsertionForProviders(100, 1, 3, true, ""),
					finalizationInsertionForProviders(99, 0, 1, false, "A")...),
			},
			{
				name:                 "mismatch-three-groups",
				consensusHashesCount: 3,
				finalizationInsertions: append(append(finalizationInsertionForProviders(100, 0, 1, true, ""),
					finalizationInsertionForProviders(100, 1, 1, false, "A")...), finalizationInsertionForProviders(100, 2, 1, false, "B")...),
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

// check this: FindRequestedBlockHash

// func TestQoS(t *testing.T) {
// 	finalizationConsensus := &FinalizationConsensus{}
// 	_ = finalizationConsensus

// 	ctx := context.Background()
// 	chainParser, _, _, closeServer, err := chainlib.CreateChainLibMocks(ctx, "LAV1", spectypes.APIInterfaceRest, func(http.ResponseWriter, *http.Request) {}, "../../")
// 	if closeServer != nil {
// 		defer closeServer()
// 	}
// 	require.NoError(t, err)
// 	require.NotNil(t, chainParser)
// 	epoch := uint64(200)
// 	finalizationConsensus.NewEpoch(epoch)
// 	consumerSessionsWithProvider := lavasession.ConsumerSessionsWithProvider{
// 		PublicLavaAddress: "",
// 		Endpoints:         []*lavasession.Endpoint{},
// 		Sessions:          map[int64]*lavasession.SingleConsumerSession{},
// 		MaxComputeUnits:   10000,
// 		UsedComputeUnits:  0,
// 		PairingEpoch:      epoch,
// 	}
// 	singleConsumerSession, _, err := consumerSessionsWithProvider.GetConsumerSessionInstanceFromEndpoint(&lavasession.Endpoint{
// 		NetworkAddress:     "",
// 		Enabled:            true,
// 		Client:             nil,
// 		ConnectionRefusals: 0,
// 	}, 1)
// 	require.NoError(t, err)
// 	require.NotNil(t, singleConsumerSession)

// 	// check updating hashes works
// 	finalizationConsensus.UpdateFinalizedHashes()

// 	expectedBH, numOfProviders := finalizationConsensus.ExpectedBlockHeight(chainParser)
// 	// pairingAddressesLen := rpccs.consumerSessionManager.GetAtomicPairingAddressesLength()
// 	// latestBlock := localRelayResult.Reply.LatestBlock
// 	// errResponse = rpccs.consumerSessionManager.OnSessionDone(singleConsumerSession, epoch, latestBlock, chainMessage.GetApi().ComputeUnits, relayLatency, singleConsumerSession.CalculateExpectedLatency(relayTimeout), expectedBH, numOfProviders, pairingAddressesLen, chainMessage.GetApi().Category.HangingApi) // session done successfully

// }
