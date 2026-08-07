package chaintracker

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/parser"
	"github.com/lavanet/lava/v5/utils/keeper"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// The two fields a getLatestBlockhash reply carries that the tracker must not
// confuse. The magnitudes mirror Solana mainnet, where accumulated skipped slots
// put the slot ~22M ahead of the block height (and lastValidBlockHeight is the
// block height at which the returned blockhash expires, ~height+150).
const (
	svmTestSlot                 = int64(436597938)
	svmTestLastValidBlockHeight = int64(414654108)
	svmTestBlockHash            = "test-hash"
	// svmTestLatestBlockhashResult is the `result` object; the reply wraps it in a
	// JSON-RPC envelope. The tracker consumes the envelope and the spec's
	// GET_BLOCKNUM directive consumes the result, so both derive from one source.
	svmTestLatestBlockhashResult = `{"context":{"slot":436597938},"value":{"blockhash":"test-hash","lastValidBlockHeight":414654108}}`
	svmTestLatestBlockhashReply  = `{"jsonrpc":"2.0","id":1,"result":` + svmTestLatestBlockhashResult + `}`
)

// svmMockChainFetcher is a minimal ChainFetcher whose CustomMessage returns a
// canned getLatestBlockhash body. fetchLatestBlockNumInner reaches the chain only
// through CustomMessage; FetchBlockHashByNum records its argument so tests can
// assert whether a hash lookup fell through to the node.
type svmMockChainFetcher struct {
	latestBlockhashResponse []byte
	hashByNumCalls          []int64
}

func (m *svmMockChainFetcher) CustomMessage(ctx context.Context, path string, data []byte, connectionType string, apiName string) ([]byte, error) {
	return m.latestBlockhashResponse, nil
}

func (m *svmMockChainFetcher) FetchLatestBlockNum(ctx context.Context) (int64, error) { return 0, nil }

func (m *svmMockChainFetcher) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	m.hashByNumCalls = append(m.hashByNumCalls, blockNum)
	return "node-hash", nil
}

func (m *svmMockChainFetcher) FetchEndpoint() lavasession.RPCProviderEndpoint {
	return lavasession.RPCProviderEndpoint{}
}

// svmMockDataFetcher stubs IChainTrackerDataFetcher. Returning zeroes keeps the
// server-memory floor at 0 so any non-negative slot passes the too-early guard.
type svmMockDataFetcher struct{}

func (svmMockDataFetcher) GetAtomicLatestBlockNum() int64 { return 0 }
func (svmMockDataFetcher) GetServerBlockMemory() uint64   { return 0 }

func newTestSVMChainTracker(t *testing.T, latestBlockhashResponse string) (*SVMChainTracker, *svmMockChainFetcher) {
	t.Helper()
	slotCache, err := ristretto.NewCache(&ristretto.Config[int64, int64]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	require.NoError(t, err)
	hashCache, err := ristretto.NewCache(&ristretto.Config[int64, string]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	require.NoError(t, err)
	chainFetcher := &svmMockChainFetcher{latestBlockhashResponse: []byte(latestBlockhashResponse)}
	return &SVMChainTracker{
		slotCache:    slotCache,
		hashCache:    hashCache,
		dataFetcher:  svmMockDataFetcher{},
		chainFetcher: chainFetcher,
	}, chainFetcher
}

// TestSVMChainTracker_TracksSlotNotLastValidBlockHeight locks the fix for #2319:
// on a getLatestBlockhash reply the tracker must expose context.slot — not
// value.lastValidBlockHeight — as the latest block.
//
// lastValidBlockHeight is the block height at which the returned blockhash
// expires. Publishing it as the chain head put RelayReply.LatestBlock ~22M below
// the slot domain that the SOLANA spec parses every numeric request argument in,
// so consistency and finalization compared values from two different domains.
func TestSVMChainTracker_TracksSlotNotLastValidBlockHeight(t *testing.T) {
	tracker, _ := newTestSVMChainTracker(t, svmTestLatestBlockhashReply)

	latestBlock, err := tracker.FetchLatestBlockNum(context.Background())
	require.NoError(t, err)

	require.Equal(t, svmTestSlot, latestBlock,
		"tracker must expose context.slot (%d) as the latest block, not value.lastValidBlockHeight (%d)", svmTestSlot, svmTestLastValidBlockHeight)
	require.NotEqual(t, svmTestLastValidBlockHeight, latestBlock,
		"tracking lastValidBlockHeight is the #2319 domain mismatch")

	// seenBlock gates hash visibility and is the value ChainTracker publishes as
	// RelayReply.LatestBlock; it must be the slot too.
	require.Equal(t, svmTestSlot, atomic.LoadInt64(&tracker.seenBlock),
		"seenBlock must be the slot, not the block height")
}

// TestSVMChainTracker_CachesHashBySlot covers the second half of #2319: the
// blockhash must be indexed by the slot it belongs to, not by its expiration
// height. Keying by lastValidBlockHeight made the cache key identify neither the
// block nor the slot the hash came from.
func TestSVMChainTracker_CachesHashBySlot(t *testing.T) {
	tracker, chainFetcher := newTestSVMChainTracker(t, svmTestLatestBlockhashReply)

	_, err := tracker.FetchLatestBlockNum(context.Background())
	require.NoError(t, err)
	// ristretto applies writes asynchronously; wait so the assertions are deterministic.
	tracker.slotCache.Wait()
	tracker.hashCache.Wait()

	cachedHash, found := tracker.hashCache.Get(svmTestSlot)
	require.True(t, found, "the hash must be cached under the slot it belongs to")
	require.Equal(t, svmTestBlockHash, cachedHash)

	_, found = tracker.hashCache.Get(svmTestLastValidBlockHeight)
	require.False(t, found, "the hash must not be cached under the blockhash expiration height")

	// A hash lookup for the tracked slot is served from cache without reaching the node.
	hash, err := tracker.FetchBlockHashByNum(context.Background(), svmTestSlot)
	require.NoError(t, err)
	require.Equal(t, svmTestBlockHash, hash)
	require.Empty(t, chainFetcher.hashByNumCalls, "a cached slot must not fall through to the node")
}

// svmSpecRPCInput is a minimal parser.RPCInput carrying only a JSON-RPC result,
// which is all a GET_BLOCKNUM result_parsing directive reads.
type svmSpecRPCInput struct {
	result json.RawMessage
}

func (i *svmSpecRPCInput) GetParams() interface{}              { return nil }
func (i *svmSpecRPCInput) GetResult() json.RawMessage          { return i.result }
func (i *svmSpecRPCInput) GetError() *rpcclient.JsonError      { return nil }
func (i *svmSpecRPCInput) GetHeaders() []pairingtypes.Metadata { return nil }
func (i *svmSpecRPCInput) GetMethod() string                   { return "" }
func (i *svmSpecRPCInput) GetID() json.RawMessage              { return nil }
func (i *svmSpecRPCInput) ParseBlock(block string) (int64, error) {
	return parser.ParseDefaultBlockParameter(block)
}

// TestSVMChainTracker_AgreesWithSpecGetBlocknum locks the invariant #2319 asks for:
// every value entering a consistency comparison must belong to one numeric domain.
//
// The other two tests pin each side separately — the tracker exposes context.slot,
// and the SOLANA spec's GET_BLOCKNUM directive parses context.slot. This one pins
// that they agree: given a single getLatestBlockhash reply, the number the tracker
// publishes as the latest block and the number the spec derives from the same reply
// must be identical. It fails if either side drifts, which neither isolated test
// would catch — and it is what makes comparing a spec-parsed request slot against
// RelayReply.LatestBlock meaningful.
func TestSVMChainTracker_AgreesWithSpecGetBlocknum(t *testing.T) {
	tracker, _ := newTestSVMChainTracker(t, svmTestLatestBlockhashReply)
	trackerLatest, err := tracker.FetchLatestBlockNum(context.Background())
	require.NoError(t, err)

	const specPath = "../../specs/mainnet-1/specs/solana.json"
	specs, err := keeper.GetAllSpecsFromFile(specPath)
	require.NoError(t, err, "must be able to load %s", specPath)
	solana, ok := specs["SOLANA"]
	require.True(t, ok, "SOLANA spec must be present in %s", specPath)

	var blockParser *spectypes.BlockParser
	for _, collection := range solana.ApiCollections {
		for _, directive := range collection.ParseDirectives {
			if directive.FunctionTag == spectypes.FUNCTION_TAG_GET_BLOCKNUM {
				bp := directive.ResultParsing
				blockParser = &bp
			}
		}
	}
	require.NotNil(t, blockParser, "SOLANA spec must define a GET_BLOCKNUM parse directive")

	specLatest := parser.ParseBlockFromReply(&svmSpecRPCInput{result: json.RawMessage(svmTestLatestBlockhashResult)}, *blockParser, nil).GetBlock()

	require.Equal(t, specLatest, trackerLatest,
		"the tracker's latest block (%d) and the spec's GET_BLOCKNUM (%d) must be the same number for the same reply", trackerLatest, specLatest)
	require.Equal(t, svmTestSlot, trackerLatest, "both must be context.slot")
}
