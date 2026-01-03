package lavasession

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
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
	AllowInsecureConnectionToProvidersFlag     = "allow-insecure-provider-dialing"
	AllowGRPCCompressionFlag                   = "enable-application-level-compression"
	MaximumStreamsOverASingleConnectionFlag    = "maximum-streams-per-connection"
	DefaultMaximumStreamsOverASingleConnection = 100
	WeightMultiplierForStaticProviders         = 10
)

var (
	AllowInsecureConnectionToProviders                   = false
	AllowGRPCCompressionForConsumerProviderCommunication = false
	MaximumStreamsOverASingleConnection                  = uint64(DefaultMaximumStreamsOverASingleConnection)
)

type UsedProvidersInf interface {
	RemoveUsed(providerAddress string, routerKey RouterKey, err error)
	TryLockSelection(context.Context) error
	AddUsed(ConsumerSessionsMap, error)
	GetUnwantedProvidersToSend(RouterKey) map[string]struct{}
	AddUnwantedAddresses(address string, routerKey RouterKey)
	CurrentlyUsed() int
}

type SessionInfo struct {
	Session           *SingleConsumerSession
	StakeSize         sdk.Coin
	QoSSummaryResult  sdk.Dec // using ComputeQoS to get the total QOS
	Epoch             uint64
	ReportedProviders []*pairingtypes.ReportedProvider
}

type ConsumerSessionsMap map[string]*SessionInfo

type ProviderOptimizer interface {
	AppendProbeRelayData(providerAddress string, latency time.Duration, success bool)
	AppendRelayFailure(providerAddress string)
	AppendRelayData(providerAddress string, latency time.Duration, cu, syncBlock uint64)
	ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string)
	ChooseProviderWithStats(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, stats *provideroptimizer.SelectionStats)
	ChooseBestProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string)
	ChooseBestProviderWithStats(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, stats *provideroptimizer.SelectionStats)
	GetReputationReportForProvider(string) (*pairingtypes.QualityOfServiceReport, time.Time)
	Strategy() provideroptimizer.Strategy
	UpdateWeights(map[string]int64, uint64)
}

type ignoredProviders struct {
	providers    map[string]struct{}
	currentEpoch uint64
}

type DataReliabilitySession struct {
	SingleConsumerSession *SingleConsumerSession
	Epoch                 uint64
	ProviderPublicAddress string
	UniqueIdentifier      bool
}

type EndpointConnection struct {
	Client                              pairingtypes.RelayerClient
	connection                          *grpc.ClientConn
	numberOfSessionsUsingThisConnection uint64
	// blockListed - currently unused, use it carefully as it will block this provider's endpoint until next epoch without forgiveness.
	// Can be used in cases of data reliability, self provider conflict etc..
	blockListed atomic.Bool
	lbUniqueId  string
	// In case we got disconnected, we cant reconnect as we might lose stickiness
	// with the provider, if its using a load balancer
	disconnected bool
}

func (ec *EndpointConnection) GetLbUniqueId() string {
	return ec.lbUniqueId
}

func (ec *EndpointConnection) addSessionUsingConnection() {
	atomic.AddUint64(&ec.numberOfSessionsUsingThisConnection, 1)
}

func (ec *EndpointConnection) decreaseSessionUsingConnection() {
	for {
		knownValue := ec.getNumberOfLiveSessionsUsingThisConnection()
		if knownValue >= 1 {
			swapped := atomic.CompareAndSwapUint64(&ec.numberOfSessionsUsingThisConnection, knownValue, knownValue-1)
			if swapped {
				return
			}
		} else {
			utils.LavaFormatError("decreaseSessionUsingConnection, Value below 1 is stored in numberOfSessionsUsingThisConnection. it must always be above 1", nil)
			return
		}
	}
}

func (ec *EndpointConnection) getNumberOfLiveSessionsUsingThisConnection() uint64 {
	return atomic.LoadUint64(&ec.numberOfSessionsUsingThisConnection)
}

type EndpointAndChosenConnection struct {
	endpoint                 *Endpoint
	chosenEndpointConnection *EndpointConnection
}

type Endpoint struct {
	NetworkAddress     string // change at the end to NetworkAddress
	Enabled            bool
	Connections        []*EndpointConnection
	ConnectionRefusals uint64
	Addons             map[string]struct{}
	Extensions         map[string]struct{}
	Geolocation        planstypes.Geolocation
}

func (e *Endpoint) CheckSupportForServices(addon string, extensions []string) (supported bool) {
	if addon != "" {
		if _, ok := e.Addons[addon]; !ok {
			return false
		}
	}
	for _, extension := range extensions {
		if extension == "" {
			continue
		}
		if _, ok := e.Extensions[extension]; !ok {
			return false
		}
	}
	return true
}

type SessionWithProvider struct {
	SessionsWithProvider *ConsumerSessionsWithProvider
	CurrentEpoch         uint64
	retryConnecting      bool
}

type SessionWithProviderMap map[string]*SessionWithProvider // key is the provider address

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
	return retStr
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
	StaticProvider                            bool
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

func (cswp *ConsumerSessionsWithProvider) IsSupportingExtensions(extensions []string, ctx context.Context) bool {
	cswp.Lock.RLock()
	defer cswp.Lock.RUnlock()

	// Debug logging for archive extension filtering
	if len(extensions) > 0 {
		utils.LavaFormatTrace("[Archive Debug] Checking extensions support",
			utils.LogAttr("providerAddress", cswp.PublicLavaAddress),
			utils.LogAttr("requestedExtensions", extensions),
			utils.LogAttr("endpointExtensions", cswp.Endpoints),
			utils.LogAttr("GUID", ctx))
	}

endpointLoop:
	for _, endpoint := range cswp.Endpoints {
		for _, extension := range extensions {
			if _, ok := endpoint.Extensions[extension]; !ok {
				// doesn't support the extension required, continue to next endpoint
				utils.LavaFormatTrace("[Archive Debug] Extension not supported",
					utils.LogAttr("providerAddress", cswp.PublicLavaAddress),
					utils.LogAttr("extension", extension),
					utils.LogAttr("endpointExtensions", endpoint.Extensions),
					utils.LogAttr("GUID", ctx))
				continue endpointLoop
			}
		}
		// get here only if all extensions are supported in the endpoint
		utils.LavaFormatTrace("[Archive Debug] All extensions supported",
			utils.LogAttr("providerAddress", cswp.PublicLavaAddress),
			utils.LogAttr("extensions", extensions),
			utils.LogAttr("GUID", ctx))
		return true
	}

	utils.LavaFormatTrace("[Archive Debug] No endpoint supports all extensions",
		utils.LogAttr("providerAddress", cswp.PublicLavaAddress),
		utils.LogAttr("extensions", extensions),
		utils.LogAttr("GUID", ctx))
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

func (cswp *ConsumerSessionsWithProvider) ConnectRawClientWithTimeout(ctx context.Context, addr string) (pairingtypes.RelayerClient, *grpc.ClientConn, error) {
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
	return c, conn, nil
}

func (cswp *ConsumerSessionsWithProvider) GetConsumerSessionInstanceFromEndpoint(endpointConnection *EndpointConnection, numberOfResets uint64, qosManager *qos.QoSManager) (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	// TODO: validate that the endpoint even belongs to the ConsumerSessionsWithProvider and is enabled.

	// Multiply numberOfReset +1 by MaxAllowedBlockListedSessionPerProvider as every reset needs to allow more blocked sessions allowed.
	maximumBlockedSessionsAllowed := uint64(utils.Min(MaxSessionsAllowedPerProvider, GetMaxAllowedBlockListedSessionPerProvider()*(int(numberOfResets)+1))) // +1 as we start from 0
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()

	// try to lock an existing session, if can't create a new one
	var numberOfBlockedSessions uint64 = 0
	for sessionID, session := range cswp.Sessions {
		if sessionID == DataReliabilitySessionId {
			continue // we cant use the data reliability session. which is located at key DataReliabilitySessionId
		}
		if session.EndpointConnection != endpointConnection {
			// skip sessions that don't belong to the active connection
			continue
		}
		blocked, ok := session.TryUseSession()
		if ok {
			return session, cswp.PairingEpoch, nil
		}
		if blocked {
			numberOfBlockedSessions += 1 // increase the number of blocked sessions so we can block this provider is too many are blocklisted
		}

		// this must come after the TryUseSession, as we need to check if we reached the maximum number of blocked sessions allowed.
		if numberOfBlockedSessions >= maximumBlockedSessionsAllowed {
			return nil, 0, MaximumNumberOfBlockListedSessionsError
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
		SessionId:          randomSessionId,
		Parent:             cswp,
		EndpointConnection: endpointConnection,
		StaticProvider:     cswp.StaticProvider,
		routerKey:          NewRouterKey(nil),
		epoch:              cswp.PairingEpoch,
		QoSManager:         qosManager,
	}

	consumerSession.TryUseSession()                            // we must lock the session so other requests wont get it.
	cswp.Sessions[consumerSession.SessionId] = consumerSession // applying the session to the pool of sessions.
	utils.LavaFormatTrace("GetConsumerSessionInstanceFromEndpoint returning session", utils.LogAttr("provider", cswp.PublicLavaAddress), utils.LogAttr("pairingEpoch", cswp.PairingEpoch), utils.LogAttr("sessionId", consumerSession.SessionId))
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
func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromConsumerSessionWithProvider(ctx context.Context, retryDisabledEndpoints bool, getAllEndpoints bool, addon string, extensionNames []string) (connected bool, endpointsList []*EndpointAndChosenConnection, providerAddress string, err error) {
	getConnectionFromConsumerSessionsWithProvider := func(ctx context.Context) (connected bool, endpointPtr []*EndpointAndChosenConnection, allDisabled bool) {
		endpoints := make([]*EndpointAndChosenConnection, 0)
		cswp.Lock.Lock()
		defer cswp.Lock.Unlock()
		for idx, endpoint := range cswp.Endpoints {
			// retryDisabledEndpoints will attempt to reconnect to the provider even though we have disabled the endpoint
			// this is used on a routine that tries to reconnect to a provider that has been disabled due to being unable to connect to it.
			if !retryDisabledEndpoints && !endpoint.Enabled {
				continue
			}
			if retryDisabledEndpoints {
				utils.LavaFormatDebug("retrying to connect to disabled endpoint", utils.LogAttr("endpoint", endpoint.NetworkAddress), utils.LogAttr("provider", cswp.PublicLavaAddress), utils.LogAttr("GUID", ctx))
			}

			// check endpoint supports the requested addons
			supported := endpoint.CheckSupportForServices(addon, extensionNames)
			if !supported {
				continue
			}
			// return
			connectEndpoint := func(cswp *ConsumerSessionsWithProvider, ctx context.Context, endpoint *Endpoint) (endpointConnection_ *EndpointConnection, connected_ bool) {
				// Clean up dead connections before iterating to prevent accumulation
				cleanedConnections := make([]*EndpointConnection, 0, len(endpoint.Connections))
				deadConnectionCount := 0
				for _, conn := range endpoint.Connections {
					// Only keep connections that are:
					// 1. Not marked as disconnected
					// 2. Still have a valid connection object
					// 3. Not in Shutdown state
					if conn.connection != nil &&
						!conn.disconnected &&
						conn.connection.GetState() != connectivity.Shutdown {
						cleanedConnections = append(cleanedConnections, conn)
					} else {
						deadConnectionCount++
						// Log cleanup for visibility
						utils.LavaFormatDebug("Cleaning up dead connection",
							utils.LogAttr("provider", cswp.PublicLavaAddress),
							utils.LogAttr("endpoint", endpoint.NetworkAddress),
							utils.LogAttr("reason", func() string {
								if conn.disconnected {
									return "marked disconnected"
								} else if conn.connection == nil {
									return "nil connection"
								} else {
									return "shutdown state"
								}
							}()),
							utils.LogAttr("GUID", ctx))
					}
				}

				// Update endpoint connections with cleaned list
				if deadConnectionCount > 0 {
					endpoint.Connections = cleanedConnections
					utils.LavaFormatDebug("Cleaned up dead connections",
						utils.LogAttr("provider", cswp.PublicLavaAddress),
						utils.LogAttr("endpoint", endpoint.NetworkAddress),
						utils.LogAttr("removedCount", deadConnectionCount),
						utils.LogAttr("remainingCount", len(cleanedConnections)),
						utils.LogAttr("GUID", ctx))
				}

				for _, endpointConnection := range endpoint.Connections {
					// If connection is active and we don't have more than maximumStreamsOverASingleConnection sessions using it already,
					// and it didn't disconnect before. Use it.
					if endpointConnection.Client != nil && endpointConnection.connection != nil && !endpointConnection.disconnected {
						// Check if the endpoint is not blocked
						if endpointConnection.blockListed.Load() {
							utils.LavaFormatDebug("Skipping provider's endpoint as its block listed", utils.LogAttr("address", endpoint.NetworkAddress), utils.LogAttr("PublicLavaAddress", cswp.PublicLavaAddress), utils.LogAttr("GUID", ctx))
							continue
						}
						connectionState := endpointConnection.connection.GetState()
						// Check Disconnections
						if connectionState == connectivity.Shutdown { // || connectionState == connectivity.Idle
							// We got disconnected, we can't use this connection anymore.
							endpointConnection.disconnected = true
							continue
						}
						// Check if we can use the connection later.
						if connectionState == connectivity.TransientFailure || connectionState == connectivity.Connecting {
							continue
						}
						// Check we didn't reach the maximum streams per connection.
						if endpointConnection.getNumberOfLiveSessionsUsingThisConnection() < MaximumStreamsOverASingleConnection {
							return endpointConnection, true
						}
					}
				}
				client, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.NetworkAddress)
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatInfo("error connecting to provider",
						utils.LogAttr("err", err),
						utils.LogAttr("provider endpoint", endpoint.NetworkAddress),
						utils.LogAttr("providerName", cswp.PublicLavaAddress),
						utils.LogAttr("endpoint", endpoint),
						utils.LogAttr("refusals", endpoint.ConnectionRefusals),
						utils.LogAttr("GUID", ctx),
					)

					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttempts {
						endpoint.Enabled = false
						utils.LavaFormatWarning("disabling provider endpoint for the duration of current epoch.", nil,
							utils.LogAttr("Endpoint", endpoint.NetworkAddress),
							utils.LogAttr("address", cswp.PublicLavaAddress),
							utils.LogAttr("GUID", ctx),
						)
					}
					return nil, false
				}
				endpoint.ConnectionRefusals = 0
				newConnection := &EndpointConnection{connection: conn, Client: client, lbUniqueId: strconv.FormatUint(utils.GenerateUniqueIdentifier(), 10)}
				endpoint.Connections = append(endpoint.Connections, newConnection)
				return newConnection, true
			}

			endpointConnection, connected_ := connectEndpoint(cswp, ctx, endpoint)
			if !connected_ {
				continue
			}
			cswp.Endpoints[idx].Enabled = true // return enabled once we successfully reconnect
			// successful new connection add to endpoints list
			endpoints = append(endpoints, &EndpointAndChosenConnection{endpoint: endpoint, chosenEndpointConnection: endpointConnection})
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
			utils.LogAttr("providerName", cswp.PublicLavaAddress),
			utils.LogAttr("GUID", ctx),
		)
		// report provider.
		return connected, endpointsList, cswp.PublicLavaAddress, AllProviderEndpointsDisabledError
	}

	return connected, endpointsList, cswp.PublicLavaAddress, nil
}

func CalcWeightsByStake(providers map[uint64]*ConsumerSessionsWithProvider) (weights map[string]int64) {
	weights = make(map[string]int64)
	staticProviders := make([]*ConsumerSessionsWithProvider, 0)
	maxWeight := int64(1)
	for _, cswp := range providers {
		if cswp.StaticProvider {
			staticProviders = append(staticProviders, cswp)
			continue
		}
		stakeAmount := cswp.getProviderStakeSize().Amount
		stake := int64(10) // defaults to 10 if stake isn't set
		if !stakeAmount.IsNil() && stakeAmount.IsInt64() {
			stake = stakeAmount.Int64()
		}
		if stake > maxWeight {
			maxWeight = stake
		}
		weights[cswp.PublicLavaAddress] = stake
	}
	for _, cswp := range staticProviders {
		weights[cswp.PublicLavaAddress] = maxWeight * WeightMultiplierForStaticProviders
	}
	return weights
}
