package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavaprotocol"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/rpcprovider"
	"github.com/lavanet/lava/protocol/rpcprovider/reliabilitymanager"
	"github.com/lavanet/lava/protocol/statetracker/updaters"
	"github.com/lavanet/lava/utils"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	plantypes "github.com/lavanet/lava/x/plans/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type TxConflictDetectionMock func(context.Context, *conflicttypes.FinalizationConflict, *conflicttypes.ResponseConflict, common.ConflictHandlerInterface) error

type mockConsumerStateTracker struct {
	txConflictDetectionMock TxConflictDetectionMock
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

func (m *mockConsumerStateTracker) SetTxConflictDetectionWrapper(txConflictDetectionWrapper TxConflictDetectionMock) {
	m.txConflictDetectionMock = txConflictDetectionWrapper
}

func (m *mockConsumerStateTracker) TxConflictDetection(ctx context.Context, finalizationConflict *conflicttypes.FinalizationConflict, responseConflict *conflicttypes.ResponseConflict, conflictHandler common.ConflictHandlerInterface) error {
	if m.txConflictDetectionMock != nil {
		return m.txConflictDetectionMock(ctx, finalizationConflict, responseConflict, conflictHandler)
	}
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

type ReplySetter struct {
	status       int
	replyDataBuf []byte
	handler      func([]byte, http.Header) ([]byte, int)
}

type mockProviderStateTracker struct {
	consumerAddressForPairing string
	averageBlockTime          time.Duration
}

func (m *mockProviderStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator updaters.VersionValidationInf) {
}

func (m *mockProviderStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	return nil
}

func (m *mockProviderStateTracker) RegisterForSpecVerifications(ctx context.Context, specVerifier updaters.SpecVerifier, chainId string) error {
	return nil
}

func (m *mockProviderStateTracker) RegisterReliabilityManagerForVoteUpdates(ctx context.Context, voteUpdatable updaters.VoteUpdatable, endpointP *lavasession.RPCProviderEndpoint) {
}

func (m *mockProviderStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable updaters.EpochUpdatable) {
}

func (m *mockProviderStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	return nil
}

func (m *mockProviderStateTracker) TxRelayPayment(ctx context.Context, relayRequests []*pairingtypes.RelaySession, description string, latestBlocks []*pairingtypes.LatestBlockReport) error {
	return nil
}

func (m *mockProviderStateTracker) SendVoteReveal(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return nil
}

func (m *mockProviderStateTracker) SendVoteCommitment(voteID string, vote *reliabilitymanager.VoteData, specID string) error {
	return nil
}

func (m *mockProviderStateTracker) LatestBlock() int64 {
	return 1000
}

func (m *mockProviderStateTracker) GetMaxCuForUser(ctx context.Context, consumerAddress, chainID string, epocu uint64) (maxCu uint64, err error) {
	return 10000, nil
}

func (m *mockProviderStateTracker) VerifyPairing(ctx context.Context, consumerAddress, providerAddress string, epoch uint64, chainID string) (valid bool, total int64, projectId string, err error) {
	return true, 10000, m.consumerAddressForPairing, nil
}

func (m *mockProviderStateTracker) GetEpochSize(ctx context.Context) (uint64, error) {
	return 30, nil
}

func (m *mockProviderStateTracker) EarliestBlockInMemory(ctx context.Context) (uint64, error) {
	return 100, nil
}

func (m *mockProviderStateTracker) RegisterPaymentUpdatableForPayments(ctx context.Context, paymentUpdatable updaters.PaymentUpdatable) {
}

func (m *mockProviderStateTracker) GetRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return 1000, nil
}

func (m *mockProviderStateTracker) GetEpochSizeMultipliedByRecommendedEpochNumToCollectPayment(ctx context.Context) (uint64, error) {
	return 30000, nil
}

func (m *mockProviderStateTracker) GetProtocolVersion(ctx context.Context) (*updaters.ProtocolVersionResponse, error) {
	return &updaters.ProtocolVersionResponse{
		Version:     &protocoltypes.Version{},
		BlockNumber: "",
	}, nil
}

func (m *mockProviderStateTracker) GetVirtualEpoch(epoch uint64) uint64 {
	return 0
}

func (m *mockProviderStateTracker) GetAverageBlockTime() time.Duration {
	return m.averageBlockTime
}

type MockChainFetcher struct {
	latestBlock int64
	blockHashes []*chaintracker.BlockStore
	mutex       sync.Mutex
	fork        string
	callBack    func()
}

func (mcf *MockChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{}
}

func (mcf *MockChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	if mcf.callBack != nil {
		mcf.callBack()
	}
	return mcf.latestBlock, nil
}

func (mcf *MockChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	for _, blockStore := range mcf.blockHashes {
		if blockStore.Block == blockNum {
			return blockStore.Hash, nil
		}
	}
	return "", fmt.Errorf("invalid block num requested %d, latestBlockSaved: %d, MockChainFetcher blockHashes: %+v", blockNum, mcf.latestBlock, mcf.blockHashes)
}

func (mcf *MockChainFetcher) FetchChainID(ctx context.Context) (string, string, error) {
	return "", "", utils.LavaFormatError("FetchChainID not supported for lava chain fetcher", nil)
}

func (mcf *MockChainFetcher) hashKey(latestBlock int64) string {
	return "stubHash-" + strconv.FormatInt(latestBlock, 10) + mcf.fork
}

func (mcf *MockChainFetcher) IsCorrectHash(hash string, hashBlock int64) bool {
	return hash == mcf.hashKey(hashBlock)
}

func (mcf *MockChainFetcher) AdvanceBlock() int64 {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	mcf.latestBlock += 1
	newHash := mcf.hashKey(mcf.latestBlock)
	mcf.blockHashes = append(mcf.blockHashes[1:], &chaintracker.BlockStore{Block: mcf.latestBlock, Hash: newHash})
	return mcf.latestBlock
}

func (mcf *MockChainFetcher) SetBlock(latestBlock int64) {
	mcf.latestBlock = latestBlock
	newHash := mcf.hashKey(mcf.latestBlock)
	mcf.blockHashes = append(mcf.blockHashes, &chaintracker.BlockStore{Block: latestBlock, Hash: newHash})
}

func (mcf *MockChainFetcher) Fork(fork string) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	if mcf.fork == fork {
		// nothing to do
		return
	}
	mcf.fork = fork
	for _, blockStore := range mcf.blockHashes {
		blockStore.Hash = mcf.hashKey(blockStore.Block)
	}
}

func (mcf *MockChainFetcher) Shrink(newSize int) {
	mcf.mutex.Lock()
	defer mcf.mutex.Unlock()
	currentSize := len(mcf.blockHashes)
	if currentSize <= newSize {
		return
	}
	newHashes := make([]*chaintracker.BlockStore, newSize)
	copy(newHashes, mcf.blockHashes[currentSize-newSize:])
}

func NewMockChainFetcher(startBlock, blocksToSave int64, callback func()) *MockChainFetcher {
	mockCHainFetcher := MockChainFetcher{callBack: callback}
	for i := int64(0); i < blocksToSave; i++ {
		mockCHainFetcher.SetBlock(startBlock + i)
	}
	return &mockCHainFetcher
}

type uniqueAddressGenerator struct {
	seed int
	lock sync.Mutex
}

func (ug *uniqueAddressGenerator) GetAddress() string {
	ug.lock.Lock()
	defer ug.lock.Unlock()
	ug.seed++
	if ug.seed < 100 {
		return "localhost:111" + strconv.Itoa(ug.seed)
	}
	return "localhost:11" + strconv.Itoa(ug.seed)
}

func (ug *uniqueAddressGenerator) GetUnixSocketAddress() string {
	ug.lock.Lock()
	defer ug.lock.Unlock()
	ug.seed++
	if ug.seed < 100 {
		return filepath.Join("/tmp", "unix:"+strconv.Itoa(ug.seed)+".sock")
	}
	return filepath.Join("/tmp", "unix:"+strconv.Itoa(ug.seed)+".sock")
}

type GetLatestBlockDataWrapper func(rpcprovider.ReliabilityManagerInf, int64, int64, int64) (int64, []*chaintracker.BlockStore, time.Time, error)

type MockReliabilityManager struct {
	ReliabilityManager        rpcprovider.ReliabilityManagerInf
	getLatestBlockDataWrapper GetLatestBlockDataWrapper
}

func NewMockReliabilityManager(reliabilityManager rpcprovider.ReliabilityManagerInf) *MockReliabilityManager {
	return &MockReliabilityManager{
		ReliabilityManager: reliabilityManager,
	}
}

func (mrm *MockReliabilityManager) SetGetLatestBlockDataWrapper(wrapper GetLatestBlockDataWrapper) {
	mrm.getLatestBlockDataWrapper = wrapper
}

func (mrm *MockReliabilityManager) GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	if mrm.getLatestBlockDataWrapper != nil {
		return mrm.getLatestBlockDataWrapper(mrm.ReliabilityManager, fromBlock, toBlock, specificBlock)
	}
	return mrm.ReliabilityManager.GetLatestBlockData(fromBlock, toBlock, specificBlock)
}

func (mrm *MockReliabilityManager) GetLatestBlockNum() (int64, time.Time) {
	return mrm.ReliabilityManager.GetLatestBlockNum()
}
