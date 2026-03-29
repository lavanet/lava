package types

import (
	"context"
	"time"

	grpc1 "github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc"
)

// EndpointService describes a single service interface offered on an endpoint.
type EndpointService struct {
	ApiInterface string `json:"api_interface"`
	Addon        string `json:"addon"`
	Extension    string `json:"extension"`
}

func (e *EndpointService) GetApiInterface() string {
	if e == nil {
		return ""
	}
	return e.ApiInterface
}

func (e *EndpointService) GetAddon() string {
	if e == nil {
		return ""
	}
	return e.Addon
}

func (e *EndpointService) GetExtension() string {
	if e == nil {
		return ""
	}
	return e.Extension
}

// Endpoint represents a single network endpoint exposed by a provider for a
// specific chain and set of API interfaces.
type Endpoint struct {
	IPPORT        string   `json:"ip_port"`
	Geolocation   int32    `json:"geolocation"`
	Addons        []string `json:"addons"`
	ApiInterfaces []string `json:"api_interfaces"`
	Extensions    []string `json:"extensions"`
}

// StakeEntry holds the on-chain staking record for a provider.
type StakeEntry struct {
	// Stake is the provider's self-delegation coin (denom + amount).
	Stake Coin `json:"stake"`
	// DelegateTotal is the total delegated stake to this provider.
	DelegateTotal Coin `json:"delegate_total"`
	// Address is the provider's account address string.
	Address string `json:"address"`
	// Vault is an optional vault address that owns the stake.
	Vault string `json:"vault"`
	// Chain is the spec / chain ID this entry belongs to.
	Chain string `json:"chain"`
	// Geolocation bitmask for this stake entry.
	Geolocation int32 `json:"geolocation"`
	// Endpoints lists all network endpoints the provider exposes.
	Endpoints []Endpoint `json:"endpoints"`
	// StakeAppliedBlock is the first block at which this stake is active.
	// A value greater than the current epoch means the provider is frozen.
	StakeAppliedBlock uint64 `json:"stake_applied_block"`
	// Jails is the cumulative number of times this provider has been jailed.
	Jails uint64 `json:"jails"`
	// JailEndTime is the unix timestamp (seconds) at which the current jail expires.
	JailEndTime int64 `json:"jail_end_time"`
}

// GetEndpoints returns the endpoint slice, never nil.
func (s *StakeEntry) GetEndpoints() []Endpoint {
	if s == nil {
		return nil
	}
	return s.Endpoints
}

// IsJailed returns true when the provider is currently jailed at the given unix
// timestamp.
func (s *StakeEntry) IsJailed(nowUnix int64) bool {
	if s == nil {
		return false
	}
	return s.JailEndTime > nowUnix
}

// IsFrozen returns true when StakeAppliedBlock is set to the sentinel maximum
// value that marks a provider as administratively frozen.
func (s *StakeEntry) IsFrozen() bool {
	if s == nil {
		return false
	}
	// The sentinel value used by the on-chain module to mark a frozen provider.
	const frozenProviderStartBlock = ^uint64(0) // max uint64
	return s.StakeAppliedBlock == frozenProviderStartBlock
}

// IsAddressVaultOrProvider returns true when addr matches either the provider
// address or the vault address on this entry.
func (s *StakeEntry) IsAddressVaultOrProvider(addr string) bool {
	if s == nil {
		return false
	}
	return s.Address == addr || s.Vault == addr
}

// ---------------------------------------------------------------------------
// Coin is a minimal stand-in for sdk.Coin that carries only what the protocol
// layer reads: a string denomination and an integer amount.
// ---------------------------------------------------------------------------

// Int wraps a simple big-integer value (stored as int64 for in-process use).
// This mirrors the subset of math.Int operations actually used in the protocol.
type Int struct {
	val int64
}

// NewInt returns an Int wrapping v.
func NewInt(v int64) Int { return Int{val: v} }

// IsNil returns false; a zero-value Int is not nil.
func (i Int) IsNil() bool { return false }

// Add returns i + other.
func (i Int) Add(other Int) Int { return Int{val: i.val + other.val} }

// Int64 returns the raw int64 value.
func (i Int) Int64() int64 { return i.val }

// Coin holds a denomination and an amount.
type Coin struct {
	Denom  string `json:"denom"`
	Amount Int    `json:"amount"`
}

// ---------------------------------------------------------------------------
// EpochDetails holds on-chain epoch boundary information.
// ---------------------------------------------------------------------------

// EpochDetails is a minimal representation of the on-chain EpochDetails object.
type EpochDetails struct {
	// StartBlock is the block height at which the current epoch started.
	StartBlock uint64 `json:"start_block"`
	// EarliestStart is the earliest remembered epoch start block still in memory.
	EarliestStart uint64 `json:"earliest_start"`
}

// GetEpochDetails returns the receiver, satisfying generated-code call patterns.
func (e *EpochDetails) GetEpochDetails() *EpochDetails {
	return e
}

// ---------------------------------------------------------------------------
// Params holds epoch-storage module parameters.
// ---------------------------------------------------------------------------

// Params is a minimal representation of epochstorage module params.
type Params struct {
	// EpochBlocks is the number of blocks per epoch.
	EpochBlocks uint64 `json:"epoch_blocks"`
}

// ---------------------------------------------------------------------------
// gRPC query request / response types
// ---------------------------------------------------------------------------

// QueryParamsRequest is the request type for the epochstorage Params RPC.
type QueryParamsRequest struct{}

// QueryParamsResponse is the response type for the epochstorage Params RPC.
type QueryParamsResponse struct {
	Params Params `json:"params"`
}

// QueryGetEpochDetailsRequest is the request type for the EpochDetails RPC.
type QueryGetEpochDetailsRequest struct{}

// QueryGetEpochDetailsResponse is the response type for the EpochDetails RPC.
type QueryGetEpochDetailsResponse struct {
	EpochDetails EpochDetails `json:"epoch_details"`
}

// ---------------------------------------------------------------------------
// QueryClient interface and constructor — mirrors the protobuf-generated shape
// ---------------------------------------------------------------------------

// QueryClient is the gRPC query client interface for the epochstorage module.
type QueryClient interface {
	// Params queries the epochstorage module parameters.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// EpochDetails queries the current epoch boundary details.
	EpochDetails(ctx context.Context, in *QueryGetEpochDetailsRequest, opts ...grpc.CallOption) (*QueryGetEpochDetailsResponse, error)
}

type queryClient struct {
	cc grpc1.ClientConn
}

// NewQueryClient returns a QueryClient that dispatches over cc.
func NewQueryClient(cc grpc1.ClientConn) QueryClient {
	return &queryClient{cc}
}

const (
	epochStorageParamsRPC       = "/lavanet.lava.epochstorage.Query/Params"
	epochStorageEpochDetailsRPC = "/lavanet.lava.epochstorage.Query/EpochDetails"
)

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	// Provide a generous default timeout when the caller has not already set one.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	out := new(QueryParamsResponse)
	if err := c.cc.Invoke(ctx, epochStorageParamsRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) EpochDetails(ctx context.Context, in *QueryGetEpochDetailsRequest, opts ...grpc.CallOption) (*QueryGetEpochDetailsResponse, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	out := new(QueryGetEpochDetailsResponse)
	if err := c.cc.Invoke(ctx, epochStorageEpochDetailsRPC, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}
