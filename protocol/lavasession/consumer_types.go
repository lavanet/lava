package lavasession

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/provideroptimizer"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/rand"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type EndpointInfo struct {
	Latency  time.Duration
	Endpoint *Endpoint
}

// Slice to hold EndpointInfo
type EndpointInfoList []EndpointInfo

// Implement sort.Interface for EndpointInfoList
func (list EndpointInfoList) Len() int {
	return len(list)
}

func (list EndpointInfoList) Less(i, j int) bool {
	return list[i].Latency < list[j].Latency
}

func (list EndpointInfoList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

const (
	AllowInsecureConnectionToProvidersFlag = "allow-insecure-provider-dialing"
	AllowGRPCCompressionFlag               = "allow-grpc-compression-for-consumer-provider-communication"
)

var (
	AllowInsecureConnectionToProviders                   = false
	AllowGRPCCompressionForConsumerProviderCommunication = false
)

type UsedProvidersInf interface {
	RemoveUsed(providerAddress string, err error)
	TryLockSelection(context.Context) bool
	AddUsed(ConsumerSessionsMap, error)
	GetUnwantedProvidersToSend() map[string]struct{}
	AddUnwantedAddresses(address string)
	CurrentlyUsed() int
}

type SessionInfo struct {
	Session           *SingleConsumerSession
	StakeSize         sdk.Coin
	QoSSummeryResult  sdk.Dec // using ComputeQoS to get the total QOS
	Epoch             uint64
	ReportedProviders []*pairingtypes.ReportedProvider
}

type ConsumerSessionsMap map[string]*SessionInfo

type ProviderOptimizer interface {
	AppendProbeRelayData(providerAddress string, latency time.Duration, success bool)
	AppendRelayFailure(providerAddress string)
	AppendRelayData(providerAddress string, latency time.Duration, isHangingApi bool, cu, syncBlock uint64)
	ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64, perturbationPercentage float64) (addresses []string)
	GetExcellenceQoSReportForProvider(string) *pairingtypes.QualityOfServiceReport
	Strategy() provideroptimizer.Strategy
}

type ignoredProviders struct {
	providers    map[string]struct{}
	currentEpoch uint64
}

type QoSReport struct {
	LastQoSReport           *pairingtypes.QualityOfServiceReport
	LastExcellenceQoSReport *pairingtypes.QualityOfServiceReport
	LatencyScoreList        []sdk.Dec
	SyncScoreSum            int64
	TotalSyncScore          int64
	TotalRelays             uint64
	AnsweredRelays          uint64
}

type DataReliabilitySession struct {
	SingleConsumerSession *SingleConsumerSession
	Epoch                 uint64
	ProviderPublicAddress string
	UniqueIdentifier      bool
}

type Endpoint struct {
	NetworkAddress     string // change at the end to NetworkAddress
	Enabled            bool
	Client             *pairingtypes.RelayerClient
	connection         *grpc.ClientConn
	ConnectionRefusals uint64
	Addons             map[string]struct{}
	Extensions         map[string]struct{}
	Geolocation        planstypes.Geolocation
}

type SessionWithProvider struct {
	SessionsWithProvider *ConsumerSessionsWithProvider
	CurrentEpoch         uint64
}

type SessionWithProviderMap map[string]*SessionWithProvider

type RPCEndpoint struct {
	NetworkAddress  string `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address"` // HOST:PORT
	ChainID         string `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                      // spec chain identifier
	ApiInterface    string `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	TLSEnabled      bool   `yaml:"tls-enabled,omitempty" json:"tls-enabled,omitempty" mapstructure:"tls-enabled"`
	HealthCheckPath string `yaml:"health-check-path,omitempty" json:"health-check-path,omitempty" mapstructure:"health-check-path"` // health check status code 200 path, default is "/"
	Geolocation     uint64 `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
}

func (endpoint *RPCEndpoint) String() (retStr string) {
	retStr = endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10)
	return
}

func (rpce *RPCEndpoint) New(address, chainID, apiInterface string, geolocation uint64) *RPCEndpoint {
	// TODO: validate correct url address
	rpce.NetworkAddress = address
	rpce.ChainID = chainID
	rpce.ApiInterface = apiInterface
	rpce.Geolocation = geolocation
	return rpce
}

func (rpce *RPCEndpoint) Key() string {
	return rpce.ChainID + rpce.ApiInterface
}

type ConsumerSessionsWithProvider struct {
	Lock              sync.RWMutex
	PublicLavaAddress string
	Endpoints         []*Endpoint
	Sessions          map[int64]*SingleConsumerSession
	MaxComputeUnits   uint64
	UsedComputeUnits  uint64
	PairingEpoch      uint64
	// whether we already reported this provider this epoch, we can only report one conflict per provider per epoch
	conflictFoundAndReported uint32   // 0 == not reported, 1 == reported
	stakeSize                sdk.Coin // the stake size the provider staked

	// blocked provider recovery status if 0 currently not used, if 1 a session has tried resume communication with this provider
	// if the provider is not blocked at all this field is irrelevant
	blockedAndUsedWithChanceForRecoveryStatus uint32
}

func NewConsumerSessionWithProvider(publicLavaAddress string, pairingEndpoints []*Endpoint, maxCu uint64, epoch uint64, stakeSize sdk.Coin) *ConsumerSessionsWithProvider {
	return &ConsumerSessionsWithProvider{
		PublicLavaAddress: publicLavaAddress,
		Endpoints:         pairingEndpoints,
		Sessions:          map[int64]*SingleConsumerSession{},
		MaxComputeUnits:   maxCu,
		PairingEpoch:      epoch,
		stakeSize:         stakeSize,
	}
}

func (cswp *ConsumerSessionsWithProvider) atomicReadBlockedStatus() uint32 {
	return atomic.LoadUint32(&cswp.blockedAndUsedWithChanceForRecoveryStatus)
}

func (cswp *ConsumerSessionsWithProvider) atomicWriteBlockedStatus(status uint32) {
	atomic.StoreUint32(&cswp.blockedAndUsedWithChanceForRecoveryStatus, status) // we can only set conflict to "reported".
}

func (cswp *ConsumerSessionsWithProvider) atomicReadConflictReported() bool {
	return atomic.LoadUint32(&cswp.conflictFoundAndReported) == 1
}

func (cswp *ConsumerSessionsWithProvider) atomicWriteConflictReported() {
	atomic.StoreUint32(&cswp.conflictFoundAndReported, 1) // we can only set conflict to "reported".
}

// checking if this provider was reported this epoch already, as we can only report once per epoch
func (cswp *ConsumerSessionsWithProvider) ConflictAlreadyReported() bool {
	// returns true if reported, false if not.
	return cswp.atomicReadConflictReported()
}

// setting this provider as conflict reported.
func (cswp *ConsumerSessionsWithProvider) StoreConflictReported() {
	cswp.atomicWriteConflictReported()
}

func (cswp *ConsumerSessionsWithProvider) IsSupportingAddon(addon string) bool {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
	if addon == "" {
		return true
	}
	for _, endpoint := range cswp.Endpoints {
		if _, ok := endpoint.Addons[addon]; ok {
			return true
		}
	}
	return false
}

func (cswp *ConsumerSessionsWithProvider) IsSupportingExtensions(extensions []string) bool {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
endpointLoop:
	for _, endpoint := range cswp.Endpoints {
		for _, extension := range extensions {
			if _, ok := endpoint.Extensions[extension]; !ok {
				// doesn't support the extension required, continue to next endpoint
				continue endpointLoop
			}
		}
		// get here only if all extensions are supported in the endpoint
		return true
	}
	return false
}

func (cswp *ConsumerSessionsWithProvider) atomicReadUsedComputeUnits() uint64 {
	return atomic.LoadUint64(&cswp.UsedComputeUnits)
}

func (cswp *ConsumerSessionsWithProvider) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&cswp.PairingEpoch)
}

func (cswp *ConsumerSessionsWithProvider) getPublicLavaAddressAndPairingEpoch() (string, uint64) {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
	return cswp.PublicLavaAddress, cswp.PairingEpoch
}

// Validate the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) validateComputeUnits(cu uint64, virtualEpoch uint64) error {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
	// add additional CU for virtual epochs
	if (cswp.UsedComputeUnits + cu) > cswp.MaxComputeUnits*(virtualEpoch+1) {
		return utils.LavaFormatWarning("validateComputeUnits", MaxComputeUnitsExceededError,
			utils.LogAttr("cu", cswp.UsedComputeUnits+cu),
			utils.LogAttr("maxCu", cswp.MaxComputeUnits*(virtualEpoch+1)),
			utils.LogAttr("virtualEpoch", virtualEpoch),
		)
	}
	return nil
}

// Validate and add the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) addUsedComputeUnits(cu, virtualEpoch uint64) error {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	// add additional CU for virtual epochs
	if (cswp.UsedComputeUnits + cu) > cswp.MaxComputeUnits*(virtualEpoch+1) {
		return MaxComputeUnitsExceededError
	}
	cswp.UsedComputeUnits += cu
	return nil
}

// check whether the provider has a specific geolocation
// used to filter reports. if the provider does not have our geolocation we do not report it
func (cswp *ConsumerSessionsWithProvider) doesProviderEndpointsContainGeolocation(geolocation uint64) bool {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
	// add additional CU for virtual epochs
	for _, endpoint := range cswp.Endpoints {
		if uint64(endpoint.Geolocation) == geolocation {
			return true
		}
	}
	return false
}

// Validate and add the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) getProviderStakeSize() sdk.Coin {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()
	return cswp.stakeSize
}

// Validate and add the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) decreaseUsedComputeUnits(cu uint64) error {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if cswp.UsedComputeUnits < cu {
		return NegativeComputeUnitsAmountError
	}
	cswp.UsedComputeUnits -= cu
	return nil
}

func (cswp *ConsumerSessionsWithProvider) ConnectRawClientWithTimeout(ctx context.Context, addr string) (*pairingtypes.RelayerClient, *grpc.ClientConn, error) {
	connectCtx, cancel := context.WithTimeout(ctx, TimeoutForEstablishingAConnection)
	defer cancel()
	conn, err := ConnectGRPCClient(connectCtx, addr, AllowInsecureConnectionToProviders, false, AllowGRPCCompressionForConsumerProviderCommunication)
	if err != nil {
		return nil, nil, err
	}
	ch := make(chan bool)
	go func() {
		for {
			// Check if the connection state is not Connecting
			if conn.GetState() == connectivity.Ready {
				ch <- true
				return
			}
			// Add some delay to avoid busy-waiting
			time.Sleep(20 * time.Millisecond)
		}
	}()
	select {
	case <-connectCtx.Done():
	case <-ch:
	}
	c := pairingtypes.NewRelayerClient(conn)
	return &c, conn, nil
}

func (cswp *ConsumerSessionsWithProvider) GetConsumerSessionInstanceFromEndpoint(endpoint *Endpoint, numberOfResets uint64) (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	// TODO: validate that the endpoint even belongs to the ConsumerSessionsWithProvider and is enabled.

	// Multiply numberOfReset +1 by MaxAllowedBlockListedSessionPerProvider as every reset needs to allow more blocked sessions allowed.
	maximumBlockedSessionsAllowed := MaxAllowedBlockListedSessionPerProvider * (numberOfResets + 1) // +1 as we start from 0
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()

	// try to lock an existing session, if can't create a new one
	var numberOfBlockedSessions uint64 = 0
	for sessionID, session := range cswp.Sessions {
		if sessionID == DataReliabilitySessionId {
			continue // we cant use the data reliability session. which is located at key DataReliabilitySessionId
		}
		if session.Endpoint != endpoint {
			// skip sessions that don't belong to the active connection
			continue
		}
		if numberOfBlockedSessions >= maximumBlockedSessionsAllowed {
			return nil, 0, MaximumNumberOfBlockListedSessionsError
		}
		blocked, ok := session.TryUseSession()
		if ok {
			return session, cswp.PairingEpoch, nil
		}
		if blocked {
			numberOfBlockedSessions += 1 // increase the number of blocked sessions so we can block this provider is too many are blocklisted
		}
	}
	// No Sessions available, create a new session or return an error upon maximum sessions allowed
	if len(cswp.Sessions) > MaxSessionsAllowedPerProvider {
		return nil, 0, MaximumNumberOfSessionsExceededError
	}

	randomSessionId := int64(0)
	for randomSessionId == 0 { // we don't allow 0
		randomSessionId = rand.Int63()
	}

	consumerSession := &SingleConsumerSession{
		SessionId: randomSessionId,
		Parent:    cswp,
		Endpoint:  endpoint,
	}
	consumerSession.TryUseSession() // we must lock the session so other requests wont get it.

	cswp.Sessions[consumerSession.SessionId] = consumerSession // applying the session to the pool of sessions.
	return consumerSession, cswp.PairingEpoch, nil
}

func (cswp *ConsumerSessionsWithProvider) sortEndpointsByLatency(endpointInfos []EndpointInfo) {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()

	// validate we do not overflow no matter what.
	if len(endpointInfos) > len(cswp.Endpoints) {
		utils.LavaFormatError("Not suppose to have larger endpointInfos length than cswp.Endpoints length", nil, utils.LogAttr("endpointInfos", endpointInfos), utils.LogAttr("cswp.Endpoints", cswp.Endpoints))
		return
	}

	// endpoint infos are already sorted by the best latency endpoint
	for idx, endpoint := range endpointInfos {
		// find the endpoint, and swap if indexes do not match expected by latency
		for cswpEndpointIdx, cswpEndpoint := range cswp.Endpoints {
			if cswpEndpoint.NetworkAddress == endpoint.Endpoint.NetworkAddress {
				// found endpoint check the index location matches the order of best endpoints
				if cswpEndpointIdx == idx {
					break
				} else {
					// we need to swap the indexes of the endpoints.
					tmpEndpoint := cswp.Endpoints[idx]
					cswp.Endpoints[idx] = endpoint.Endpoint
					cswp.Endpoints[cswpEndpointIdx] = tmpEndpoint
					break
				}
			}
		}
	}
}

// fetching an endpoint from a ConsumerSessionWithProvider and establishing a connection,
// can fail without an error if trying to connect once to each endpoint but none of them are active.
func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromConsumerSessionWithProvider(ctx context.Context, retryDisabledEndpoints bool, getAllEndpoints bool) (connected bool, endpointsList []*Endpoint, providerAddress string, err error) {
	getConnectionFromConsumerSessionsWithProvider := func(ctx context.Context) (connected bool, endpointPtr []*Endpoint, allDisabled bool) {
		endpoints := make([]*Endpoint, 0)
		cswp.Lock.Lock()
		defer cswp.Lock.Unlock()
		for idx, endpoint := range cswp.Endpoints {
			// retryDisabledEndpoints will attempt to reconnect to the provider even though we have disabled the endpoint
			// this is used on a routine that tries to reconnect to a provider that has been disabled due to being unable to connect to it.
			if !retryDisabledEndpoints && !endpoint.Enabled {
				continue
			}
			connectEndpoint := func(cswp *ConsumerSessionsWithProvider, ctx context.Context, endpoint *Endpoint) (connected_ bool) {
				if endpoint.Client != nil && endpoint.connection != nil && endpoint.connection.GetState() != connectivity.Shutdown && endpoint.connection.GetState() != connectivity.Idle {
					return true
				}
				client, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.NetworkAddress)
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatInfo("error connecting to provider",
						utils.LogAttr("err", err),
						utils.LogAttr("provider endpoint", endpoint.NetworkAddress),
						utils.LogAttr("provider address", cswp.PublicLavaAddress),
						utils.LogAttr("endpoint", endpoint),
						utils.LogAttr("refusals", endpoint.ConnectionRefusals),
					)

					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttempts {
						endpoint.Enabled = false
						utils.LavaFormatWarning("disabling provider endpoint for the duration of current epoch.", nil,
							utils.LogAttr("Endpoint", endpoint.NetworkAddress),
							utils.LogAttr("address", cswp.PublicLavaAddress),
						)
					}
					return false
				}
				endpoint.ConnectionRefusals = 0
				endpoint.Client = client
				if endpoint.connection != nil {
					endpoint.connection.Close() // just to be safe
				}
				endpoint.connection = conn
				return true
			}
			endpointState := connectivity.Idle
			if endpoint.connection != nil {
				endpointState = endpoint.connection.GetState()
			}
			if endpoint.Client == nil {
				connected_ := connectEndpoint(cswp, ctx, endpoint)
				if !connected_ {
					continue
				}
			} else if endpointState == connectivity.Shutdown || endpointState == connectivity.Idle {
				// connection was shut down, so we need to create a new one
				endpoint.connection.Close()
				connected_ := connectEndpoint(cswp, ctx, endpoint)
				if !connected_ {
					continue
				}
			} else if endpointState == connectivity.TransientFailure || endpointState == connectivity.Connecting {
				// can't use this one right now, but we could in the future
				continue
			}
			cswp.Endpoints[idx] = endpoint
			cswp.Endpoints[idx].Enabled = true // return enabled once we successfully reconnect
			// successful connection add to endpoints list
			endpoints = append(endpoints, endpoint)
			if !getAllEndpoints {
				return true, endpoints, false
			}
		}

		// if we managed to get at least one endpoint we can return the list of active endpoints
		if len(endpoints) > 0 {
			return true, endpoints, false
		}

		// checking disabled endpoints, as we can disable an endpoint mid run of the previous loop, we should re test the current endpoint state
		// before verifying all are Disabled.
		allDisabled = true
		for _, endpoint := range cswp.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			// even one endpoint is enough for us to not purge.
			allDisabled = false
		}
		return false, nil, allDisabled
	}

	var allDisabled bool
	connected, endpointsList, allDisabled = getConnectionFromConsumerSessionsWithProvider(ctx)
	if allDisabled {
		utils.LavaFormatInfo("purging provider after all endpoints are disabled",
			utils.LogAttr("provider endpoints", cswp.Endpoints),
			utils.LogAttr("provider address", cswp.PublicLavaAddress),
		)
		// report provider.
		return connected, endpointsList, cswp.PublicLavaAddress, AllProviderEndpointsDisabledError
	}

	return connected, endpointsList, cswp.PublicLavaAddress, nil
}

func CalculateAvailabilityScore(qosReport *QoSReport) (downtimePercentageRet, scaledAvailabilityScoreRet sdk.Dec) {
	downtimePercentage := sdk.NewDecWithPrec(int64(qosReport.TotalRelays-qosReport.AnsweredRelays), 0).Quo(sdk.NewDecWithPrec(int64(qosReport.TotalRelays), 0))
	scaledAvailabilityScore := sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePercentage).Quo(AvailabilityPercentage))
	return downtimePercentage, scaledAvailabilityScore
}
