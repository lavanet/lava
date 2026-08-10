package rpcprovider

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v5/protocol/chainlib"
	rpcclient "github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	extensionslib "github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/chaintracker"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

var errFakeChainParserNotImplemented = errors.New("fakeChainParser: not implemented")

// sequenceChainTracker returns a deterministic latest block sequence across calls.
// It embeds DummyChainTracker to satisfy the full IChainTracker interface.
type sequenceChainTracker struct {
	*chaintracker.DummyChainTracker
	seq        []int64
	callIdx    atomic.Int32
	changeTime time.Time
}

func (sct *sequenceChainTracker) current() int64 {
	i := int(sct.callIdx.Load())
	if i < 0 {
		return sct.seq[0]
	}
	if i >= len(sct.seq) {
		return sct.seq[len(sct.seq)-1]
	}
	return sct.seq[i]
}

// GetLatestBlockNum does not advance the sequence: consistency now reads the block
// through GetWireLatestBlock and consults this only for the change time, so advancing
// here would consume the sequence twice per poll.
func (sct *sequenceChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return sct.current(), sct.changeTime
}

func (sct *sequenceChainTracker) GetAtomicLatestBlockNum() int64 {
	return sct.current()
}

// GetWireLatestBlock walks the sequence, one entry per read. This tracker models a
// non-SVM chain, where the published value and the chain position are the same number
// — but it must still be overridden, since the embedded DummyChainTracker reports
// MaxInt64 and would make every wait succeed instantly.
func (sct *sequenceChainTracker) GetWireLatestBlock() int64 {
	i := int(sct.callIdx.Add(1)) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sct.seq) {
		i = len(sct.seq) - 1
	}
	return sct.seq[i]
}

func TestHandleConsistency_HistoricalBlockNoWait(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now().Add(-time.Second))

	rpcps := &RPCProviderServer{
		chainTracker:        mockTracker,
		rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "TEST"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	latest, slept, err := rpcps.handleConsistency(
		ctx,
		500*time.Millisecond,
		2000, // seenBlock >> latest
		500,  // specific historical request (<= latest)
		time.Second,
		1,
		0,
		0,
	)

	require.NoError(t, err)
	require.Equal(t, int64(1000), latest)
	require.Equal(t, time.Duration(0), slept)
}

func TestHandleConsistency_TooNewBailsFast(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1, time.Now().Add(-time.Second))

	rpcps := &RPCProviderServer{
		chainTracker:        mockTracker,
		rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "TEST"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	latest, slept, err := rpcps.handleConsistency(
		ctx,
		time.Second,
		1000, // seenBlock
		1000, // requestBlock
		time.Second,
		1, // blockLagForQosSync (small => quick bail)
		0,
		0,
	)

	require.Error(t, err)
	require.Equal(t, int64(1), latest)
	require.Equal(t, time.Duration(0), slept)
}

func TestHandleConsistency_SleepsAndCatchesUp(t *testing.T) {
	// First call returns 1, subsequent calls return 2.
	seqTracker := &sequenceChainTracker{
		DummyChainTracker: &chaintracker.DummyChainTracker{},
		seq:               []int64{1, 2, 2, 2},
		changeTime:        time.Now().Add(-time.Second),
	}

	rpcps := &RPCProviderServer{
		chainTracker:        seqTracker,
		rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "TEST"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	latest, slept, err := rpcps.handleConsistency(
		ctx,
		20*time.Millisecond,
		2, // seenBlock
		2, // requestBlock
		100*time.Millisecond,
		100, // large lag threshold => don't bail
		0,
		0,
	)

	require.NoError(t, err)
	require.Equal(t, int64(2), latest)
	require.Greater(t, slept, time.Duration(0))
}

// splitDomainChainTracker models an SVM provider after the domain split: its chain
// position is a slot, while the value it publishes to consumers stays a block height.
// The two differ by roughly the Solana slot-vs-height gap.
type splitDomainChainTracker struct {
	*chaintracker.DummyChainTracker
	wireLatestBlock    int64
	chainPositionBlock int64
	changeTime         time.Time
}

func (sdt *splitDomainChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return sdt.chainPositionBlock, sdt.changeTime
}
func (sdt *splitDomainChainTracker) GetAtomicLatestBlockNum() int64 { return sdt.chainPositionBlock }
func (sdt *splitDomainChainTracker) GetWireLatestBlock() int64      { return sdt.wireLatestBlock }

// TestHandleConsistency_ComparesInPublishedDomain guards the quiet failure mode of the
// SVM domain split. seenBlock and requestBlock both originate from the consumer, which
// only ever saw published values, so consistency has to compare against the published
// value too.
//
// Reading the chain position instead would not error — it would silently pass every
// check, because a Solana slot sits ~22M above the block height a consumer reports.
// Consistency enforcement for Solana would switch off with nothing in the logs.
func TestHandleConsistency_ComparesInPublishedDomain(t *testing.T) {
	const (
		wireLatest    = int64(414654108) // block height, what consumers exchange
		chainPosition = int64(436597938) // slot, the tracker's internal position
	)

	newServer := func() *RPCProviderServer {
		return &RPCProviderServer{
			chainTracker: &splitDomainChainTracker{
				DummyChainTracker:  &chaintracker.DummyChainTracker{},
				wireLatestBlock:    wireLatest,
				chainPositionBlock: chainPosition,
				changeTime:         time.Now().Add(-time.Second),
			},
			rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "SOLANA"},
		}
	}

	t.Run("reports the published value, not the chain position", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Comfortably behind the published latest, so this returns immediately.
		latest, slept, err := newServer().handleConsistency(ctx, 20*time.Millisecond,
			wireLatest-10, wireLatest-10, 400*time.Millisecond, 25, 0, 1)

		require.NoError(t, err)
		require.Zero(t, slept)
		require.Equal(t, wireLatest, latest,
			"RelayReply.LatestBlock is fed from here and must stay in the published domain")
		require.NotEqual(t, chainPosition, latest)
	})

	t.Run("a seen block past the published latest is still enforced", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Above the published latest but far below the chain position: reading the
		// chain position here would return nil and skip the check entirely.
		_, _, err := newServer().handleConsistency(ctx, 20*time.Millisecond,
			wireLatest+500, wireLatest+500, 400*time.Millisecond, 25, 0, 1)

		require.Error(t, err,
			"consistency must still fire; comparing against the chain position would silently pass")
	})
}

// TestHandleConsistency_SmallGapWaitsInsteadOfBailing pins the NEAR-shaped case: the consumer's
// seenBlock is a few blocks ahead of this provider's chain tracker, but those blocks already exist
// on chain and the tracker picks them up on its next poll. The provider must spend its wait budget
// instead of bailing with ConsistencyError, even when the spec's average_block_time overstates the
// chain's real block time (which inflates probabilityBlockError to ~1 for any gap >= 2).
func TestHandleConsistency_SmallGapWaitsInsteadOfBailing(t *testing.T) {
	for _, blockGap := range []int64{2, 3} {
		seqTracker := &sequenceChainTracker{
			DummyChainTracker: &chaintracker.DummyChainTracker{},
			// first read is behind by blockGap, the next poll has caught up
			seq:        []int64{1000, 1000 + blockGap, 1000 + blockGap, 1000 + blockGap},
			changeTime: time.Now().Add(-300 * time.Millisecond),
		}

		rpcps := &RPCProviderServer{
			chainTracker:        seqTracker,
			rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "NEAR"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		latest, slept, err := rpcps.handleConsistency(
			ctx,
			time.Second,            // baseRelayTimeout: 10 CU => halfTimeLeft caps at 500ms
			1000+blockGap,          // seenBlock, already served by another provider
			spectypes.LATEST_BLOCK, // requestBlock
			1200*time.Millisecond,  // stale average_block_time (chain really runs at ~600ms)
			16,                     // allowed_block_lag_for_qos_sync
			3,
			3,
		)
		cancel()

		require.NoError(t, err, "blockGap %d must not bail, the tracker catches up on its next poll", blockGap)
		require.Equal(t, 1000+blockGap, latest)
		require.Greater(t, slept, time.Duration(0), "blockGap %d should have used the wait budget", blockGap)
	}
}

// TestHandleConsistency_NoTimeToPollBails covers the other side of the gate: when the caller leaves
// less time than a single chain tracker poll, waiting cannot help, so bail immediately.
func TestHandleConsistency_NoTimeToPollBails(t *testing.T) {
	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1000, time.Now().Add(-100*time.Millisecond))

	rpcps := &RPCProviderServer{
		chainTracker:        mockTracker,
		rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "NEAR"},
	}

	// averageBlockTime 1200ms => tracker polls at worst every 300ms; a 100ms relay budget
	// leaves halfTimeLeft of 50ms, which cannot fit a poll.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	latest, slept, err := rpcps.handleConsistency(
		ctx,
		100*time.Millisecond,
		1002,
		spectypes.LATEST_BLOCK,
		1200*time.Millisecond,
		16,
		3,
		3,
	)

	require.Error(t, err)
	require.Equal(t, int64(1000), latest)
	require.Equal(t, time.Duration(0), slept)
}

type fakeChainParser struct {
	blockLagForQosSync        int64
	averageBlockTime          time.Duration
	blockDistanceToFinalized  uint32
	blocksInFinalizationProof uint32
}

func (fcp *fakeChainParser) ParseMsg(url string, data []byte, connectionType string, metadata []pairingtypes.Metadata, extensionInfo extensionslib.ExtensionInfo) (chainlib.ChainMessage, error) {
	return nil, errFakeChainParserNotImplemented
}
func (fcp *fakeChainParser) SetSpec(spec spectypes.Spec) {}
func (fcp *fakeChainParser) ChainBlockStats() (int64, time.Duration, uint32, uint32) {
	return fcp.blockLagForQosSync, fcp.averageBlockTime, fcp.blockDistanceToFinalized, fcp.blocksInFinalizationProof
}

func (fcp *fakeChainParser) GetParsingByTag(tag spectypes.FUNCTION_TAG) (*spectypes.ParseDirective, *spectypes.ApiCollection, bool) {
	return nil, nil, false
}

func (fcp *fakeChainParser) IsTagInCollection(tag spectypes.FUNCTION_TAG, collectionKey chainlib.CollectionKey) bool {
	return false
}
func (fcp *fakeChainParser) GetAllInternalPaths() []string { return nil }
func (fcp *fakeChainParser) IsInternalPathEnabled(internalPath string, apiInterface string, addon string) bool {
	return true
}

func (fcp *fakeChainParser) CraftMessage(parser *spectypes.ParseDirective, connectionType string, craftData *chainlib.CraftData, metadata []pairingtypes.Metadata) (chainlib.ChainMessageForSend, error) {
	return nil, errFakeChainParserNotImplemented
}

func (fcp *fakeChainParser) HandleHeaders(metadata []pairingtypes.Metadata, apiCollection *spectypes.ApiCollection, headersDirection spectypes.Header_HeaderType) ([]pairingtypes.Metadata, string, []pairingtypes.Metadata) {
	return metadata, "", nil
}

func (fcp *fakeChainParser) GetVerifications(supported []string, internalPath string, apiInterface string) ([]chainlib.VerificationContainer, error) {
	return []chainlib.VerificationContainer{}, nil
}

func (fcp *fakeChainParser) SeparateAddonsExtensions(ctx context.Context, supported []string) (addons, extensions []string, err error) {
	return []string{}, []string{}, nil
}

func (fcp *fakeChainParser) SetPolicy(policy chainlib.PolicyInf, chainId string, apiInterface string) error {
	return nil
}
func (fcp *fakeChainParser) Active() bool                                     { return true }
func (fcp *fakeChainParser) Activate()                                        {}
func (fcp *fakeChainParser) UpdateBlockTime(newBlockTime time.Duration)       {}
func (fcp *fakeChainParser) GetUniqueName() string                            { return "fake" }
func (fcp *fakeChainParser) ExtensionsParser() *extensionslib.ExtensionParser { return nil }
func (fcp *fakeChainParser) ExtractDataFromRequest(*http.Request) (string, string, string, []pairingtypes.Metadata, error) {
	return "", "", "", nil, nil
}

func (fcp *fakeChainParser) SetResponseFromRelayResult(*common.RelayResult) (*http.Response, error) {
	return &http.Response{}, nil
}
func (fcp *fakeChainParser) ValidateMessage(chainlib.ChainMessage) error { return nil }
func (fcp *fakeChainParser) ParseDirectiveEnabled() bool                 { return true }

type fakeChainRouter struct{}

func (fcr *fakeChainRouter) SendNodeMsg(ctx context.Context, ch chan interface{}, chainMessage chainlib.ChainMessageForSend, extensions []string) (*chainlib.RelayReplyWrapper, string, *rpcclient.ClientSubscription, common.NodeUrl, string, error) {
	return nil, "", nil, common.NodeUrl{}, "", nil
}

func (fcr *fakeChainRouter) ExtensionsSupported(internalPath string, extensions []string) bool {
	return true
}

func TestTryRelayWithWrapper_ReturnsOnConsistencyError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMsg := chainlib.NewMockChainMessage(ctrl)

	// Setup minimal expectations for ValidateRequest + ValidateAddonsExtensions + GetRelayTimeout + ChainBlockStats.
	apiCollection := &spectypes.ApiCollection{
		CollectionData: spectypes.CollectionData{
			AddOn:        "",
			InternalPath: "p",
		},
	}
	mockMsg.EXPECT().RequestedBlock().Return(int64(1000), int64(0)).Times(1)
	mockMsg.EXPECT().GetApiCollection().Return(apiCollection).AnyTimes()
	mockMsg.EXPECT().TimeoutOverride().Return(time.Duration(0)).AnyTimes()
	mockMsg.EXPECT().GetApi().Return(&spectypes.Api{Name: "test"}).AnyTimes()

	mockTracker := NewMockChainTracker()
	mockTracker.SetLatestBlock(1, time.Now().Add(-time.Second))

	parser := &fakeChainParser{
		blockLagForQosSync: int64(1),
		averageBlockTime:   time.Second,
	}

	rpcps := &RPCProviderServer{
		chainParser:         parser,
		chainRouter:         &fakeChainRouter{},
		chainTracker:        mockTracker,
		enableConsistency:   true,
		rpcProviderEndpoint: &lavasession.RPCProviderEndpoint{ChainID: "TEST"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	request := &pairingtypes.RelayRequest{
		RelayData: &pairingtypes.RelayPrivateData{
			RequestBlock: 1000,
			SeenBlock:    1000,
			Addon:        "",
			Extensions:   []string{},
		},
	}

	consumerAddr := sdk.AccAddress([]byte("12345678901234567890"))
	reply, wrapper, err := rpcps.TryRelayWithWrapper(ctx, request, consumerAddr, mockMsg)

	require.Error(t, err)
	require.Nil(t, reply)
	require.Nil(t, wrapper)
}
