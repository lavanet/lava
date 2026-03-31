package relay

import (
	"context"
	"time"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	epochstoragetypes "github.com/lavanet/lava/v5/types/epoch"
	plantypes "github.com/lavanet/lava/v5/types/plans"
	"google.golang.org/grpc"
)

// ---------------------------------------------------------------------------
// Event constants
// ---------------------------------------------------------------------------

// RelayPaymentEventName is the Cosmos event type emitted when a relay payment
// is submitted on-chain.
const RelayPaymentEventName = "relay_payment"

// ---------------------------------------------------------------------------
// Params — pairing module parameters
// ---------------------------------------------------------------------------

// Params holds the pairing module's on-chain parameters.
type Params struct {
	// EpochBlocksOverlap is the number of blocks before the next epoch at which
	// new pairings take effect (overlap to ensure smooth transition).
	EpochBlocksOverlap uint64 `json:"epoch_blocks_overlap"`

	// RecommendedEpochNumToCollectPayment is the recommended number of epochs
	// after which a provider should collect payment.
	RecommendedEpochNumToCollectPayment uint64 `json:"recommended_epoch_num_to_collect_payment"`
}

func (p *Params) GetEpochBlocksOverlap() uint64 {
	if p != nil {
		return p.EpochBlocksOverlap
	}
	return 0
}

func (p *Params) GetRecommendedEpochNumToCollectPayment() uint64 {
	if p != nil {
		return p.RecommendedEpochNumToCollectPayment
	}
	return 0
}

// ---------------------------------------------------------------------------
// Query request / response types
// ---------------------------------------------------------------------------

// QueryParamsRequest is the request for the pairing Params RPC.
type QueryParamsRequest struct{}

// QueryParamsResponse is the response for the pairing Params RPC.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

func (r *QueryParamsResponse) GetParams() *Params {
	if r != nil {
		return &r.Params
	}
	return nil
}

// QueryProviderRequest is the request for the pairing Provider RPC.
type QueryProviderRequest struct {
	Address string `json:"address"`
}

func (r *QueryProviderRequest) GetAddress() string {
	if r != nil {
		return r.Address
	}
	return ""
}

// QueryProviderResponse is the response for the pairing Provider RPC.
type QueryProviderResponse struct {
	StakeEntries []epochstoragetypes.StakeEntry `json:"stake_entries"`
}

func (r *QueryProviderResponse) GetStakeEntries() []epochstoragetypes.StakeEntry {
	if r != nil {
		return r.StakeEntries
	}
	return nil
}

// QueryGetPairingRequest is the request for the GetPairing RPC.
type QueryGetPairingRequest struct {
	ChainID string `json:"chain_id"`
	Client  string `json:"client"`
}

func (r *QueryGetPairingRequest) GetChainID() string {
	if r != nil {
		return r.ChainID
	}
	return ""
}

func (r *QueryGetPairingRequest) GetClient() string {
	if r != nil {
		return r.Client
	}
	return ""
}

// QueryGetPairingResponse is the response for the GetPairing RPC.
type QueryGetPairingResponse struct {
	Providers          []epochstoragetypes.StakeEntry `json:"providers"`
	CurrentEpoch       uint64                         `json:"current_epoch"`
	BlockOfNextPairing uint64                         `json:"block_of_next_pairing"`
}

func (r *QueryGetPairingResponse) GetProviders() []epochstoragetypes.StakeEntry {
	if r != nil {
		return r.Providers
	}
	return nil
}

func (r *QueryGetPairingResponse) GetCurrentEpoch() uint64 {
	if r != nil {
		return r.CurrentEpoch
	}
	return 0
}

func (r *QueryGetPairingResponse) GetBlockOfNextPairing() uint64 {
	if r != nil {
		return r.BlockOfNextPairing
	}
	return 0
}

// QueryUserEntryRequest is the request for the UserEntry RPC.
type QueryUserEntryRequest struct {
	ChainID string `json:"chain_id"`
	Address string `json:"address"`
	Block   uint64 `json:"block"`
}

func (r *QueryUserEntryRequest) GetChainID() string {
	if r != nil {
		return r.ChainID
	}
	return ""
}

func (r *QueryUserEntryRequest) GetAddress() string {
	if r != nil {
		return r.Address
	}
	return ""
}

func (r *QueryUserEntryRequest) GetBlock() uint64 {
	if r != nil {
		return r.Block
	}
	return 0
}

// QueryUserEntryResponse is the response for the UserEntry RPC.
type QueryUserEntryResponse struct {
	// MaxCU is the maximum compute units available to the user this epoch.
	MaxCU uint64 `json:"max_cu"`
}

func (r *QueryUserEntryResponse) GetMaxCU() uint64 {
	if r != nil {
		return r.MaxCU
	}
	return 0
}

// QueryVerifyPairingRequest is the request for the VerifyPairing RPC.
type QueryVerifyPairingRequest struct {
	ChainID  string `json:"chain_id"`
	Client   string `json:"client"`
	Provider string `json:"provider"`
	Block    uint64 `json:"block"`
}

func (r *QueryVerifyPairingRequest) GetChainID() string {
	if r != nil {
		return r.ChainID
	}
	return ""
}

func (r *QueryVerifyPairingRequest) GetClient() string {
	if r != nil {
		return r.Client
	}
	return ""
}

func (r *QueryVerifyPairingRequest) GetProvider() string {
	if r != nil {
		return r.Provider
	}
	return ""
}

func (r *QueryVerifyPairingRequest) GetBlock() uint64 {
	if r != nil {
		return r.Block
	}
	return 0
}

// QueryVerifyPairingResponse is the response for the VerifyPairing RPC.
type QueryVerifyPairingResponse struct {
	Valid           bool   `json:"valid"`
	PairedProviders uint64 `json:"paired_providers"`
	ProjectId       string `json:"project_id"`
}

func (r *QueryVerifyPairingResponse) GetValid() bool {
	if r != nil {
		return r.Valid
	}
	return false
}

func (r *QueryVerifyPairingResponse) GetPairedProviders() uint64 {
	if r != nil {
		return r.PairedProviders
	}
	return 0
}

func (r *QueryVerifyPairingResponse) GetProjectId() string {
	if r != nil {
		return r.ProjectId
	}
	return ""
}

// QueryEffectivePolicyRequest is the request for the EffectivePolicy RPC.
type QueryEffectivePolicyRequest struct {
	Consumer string `json:"consumer"`
	SpecID   string `json:"spec_id"`
}

func (r *QueryEffectivePolicyRequest) GetConsumer() string {
	if r != nil {
		return r.Consumer
	}
	return ""
}

func (r *QueryEffectivePolicyRequest) GetSpecID() string {
	if r != nil {
		return r.SpecID
	}
	return ""
}

// QueryEffectivePolicyResponse is the response for the EffectivePolicy RPC.
type QueryEffectivePolicyResponse struct {
	Policy *plantypes.Policy `json:"policy"`
}

func (r *QueryEffectivePolicyResponse) GetPolicy() *plantypes.Policy {
	if r != nil {
		return r.Policy
	}
	return nil
}

// ---------------------------------------------------------------------------
// QueryClient interface and constructor
// ---------------------------------------------------------------------------

// QueryClient is the gRPC query client interface for the pairing module.
// It mirrors the interface shape of the deleted protobuf-generated client.
type QueryClient interface {
	// Params queries the pairing module parameters.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// Provider queries the stake entry for a given provider address.
	Provider(ctx context.Context, in *QueryProviderRequest, opts ...grpc.CallOption) (*QueryProviderResponse, error)
	// GetPairing queries the current pairing list for a consumer/chain pair.
	GetPairing(ctx context.Context, in *QueryGetPairingRequest, opts ...grpc.CallOption) (*QueryGetPairingResponse, error)
	// UserEntry queries the stake entry for a consumer.
	UserEntry(ctx context.Context, in *QueryUserEntryRequest, opts ...grpc.CallOption) (*QueryUserEntryResponse, error)
	// VerifyPairing verifies that a consumer-provider pairing is valid for an epoch.
	VerifyPairing(ctx context.Context, in *QueryVerifyPairingRequest, opts ...grpc.CallOption) (*QueryVerifyPairingResponse, error)
	// EffectivePolicy returns the effective subscription policy for a consumer.
	EffectivePolicy(ctx context.Context, in *QueryEffectivePolicyRequest, opts ...grpc.CallOption) (*QueryEffectivePolicyResponse, error)
}

type pairingQueryClient struct {
	cc grpc1.ClientConn
}

// NewQueryClient creates a QueryClient that dispatches over cc.
func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &pairingQueryClient{cc}
}

const (
	pairingParamsRPC          = "/lavanet.lava.pairing.Query/Params"
	pairingProviderRPC        = "/lavanet.lava.pairing.Query/Provider"
	pairingGetPairingRPC      = "/lavanet.lava.pairing.Query/GetPairing"
	pairingUserEntryRPC       = "/lavanet.lava.pairing.Query/UserEntry"
	pairingVerifyPairingRPC   = "/lavanet.lava.pairing.Query/VerifyPairing"
	pairingEffectivePolicyRPC = "/lavanet.lava.pairing.Query/EffectivePolicy"
)

func withDefaultTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 10*time.Second)
}

func (c *pairingQueryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryParamsResponse)
	if err := c.cc.Invoke(ctx, pairingParamsRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pairingQueryClient) Provider(ctx context.Context, in *QueryProviderRequest, opts ...grpc.CallOption) (*QueryProviderResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryProviderResponse)
	if err := c.cc.Invoke(ctx, pairingProviderRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pairingQueryClient) GetPairing(ctx context.Context, in *QueryGetPairingRequest, opts ...grpc.CallOption) (*QueryGetPairingResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryGetPairingResponse)
	if err := c.cc.Invoke(ctx, pairingGetPairingRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pairingQueryClient) UserEntry(ctx context.Context, in *QueryUserEntryRequest, opts ...grpc.CallOption) (*QueryUserEntryResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryUserEntryResponse)
	if err := c.cc.Invoke(ctx, pairingUserEntryRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pairingQueryClient) VerifyPairing(ctx context.Context, in *QueryVerifyPairingRequest, opts ...grpc.CallOption) (*QueryVerifyPairingResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryVerifyPairingResponse)
	if err := c.cc.Invoke(ctx, pairingVerifyPairingRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pairingQueryClient) EffectivePolicy(ctx context.Context, in *QueryEffectivePolicyRequest, opts ...grpc.CallOption) (*QueryEffectivePolicyResponse, error) {
	ctx, cancel := withDefaultTimeout(ctx)
	defer cancel()
	out := new(QueryEffectivePolicyResponse)
	if err := c.cc.Invoke(ctx, pairingEffectivePolicyRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// MsgRelayPayment — minimal stub for CLI tooling
// ---------------------------------------------------------------------------

// MsgRelayPayment is a minimal representation of the on-chain relay payment
// message, used by CLI tooling to inspect transaction contents.
type MsgRelayPayment struct {
	// Relays contains the relay sessions claimed for payment.
	Relays []*RelaySession `json:"relays"`
	// Creator is the provider's account address.
	Creator string `json:"creator"`
}

func (m *MsgRelayPayment) GetRelays() []*RelaySession {
	if m != nil {
		return m.Relays
	}
	return nil
}

func (m *MsgRelayPayment) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}
