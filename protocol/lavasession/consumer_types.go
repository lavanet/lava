package lavasession

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const AllowInsecureConnectionToProvidersFlag = "allow-insecure-provider-dialing"

var AllowInsecureConnectionToProviders = false

type SessionInfo struct {
	Session           *SingleConsumerSession
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

type SingleConsumerSession struct {
	CuSum                       uint64
	LatestRelayCu               uint64 // set by GetSessions cuNeededForSession
	QoSInfo                     QoSReport
	SessionId                   int64
	Client                      *ConsumerSessionsWithProvider
	lock                        utils.LavaMutex
	RelayNum                    uint64
	LatestBlock                 int64
	Endpoint                    *Endpoint
	BlockListed                 bool   // if session lost sync we blacklist it.
	ConsecutiveNumberOfFailures uint64 // number of times this session has failed
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
}

type SessionWithProvider struct {
	SessionsWithProvider *ConsumerSessionsWithProvider
	CurrentEpoch         uint64
}

type SessionWithProviderMap map[string]*SessionWithProvider

type RPCEndpoint struct {
	NetworkAddress string `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address"` // HOST:PORT
	ChainID        string `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                      // spec chain identifier
	ApiInterface   string `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64 `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
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
	Lock              utils.LavaMutex
	PublicLavaAddress string
	Endpoints         []*Endpoint
	Sessions          map[int64]*SingleConsumerSession
	MaxComputeUnits   uint64
	UsedComputeUnits  uint64
	PairingEpoch      uint64
	// whether we already reported this provider this epoch, we can only report one conflict per provider per epoch
	conflictFoundAndReported uint32 // 0 == not reported, 1 == reported
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
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
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
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
endpointLoop:
	for _, endpoint := range cswp.Endpoints {
		for _, extension := range extensions {
			if _, ok := endpoint.Extensions[extension]; !ok {
				// doesn;t support the extension required, continue to next endpoint
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

// verify data reliability session exists or not
func (cswp *ConsumerSessionsWithProvider) verifyDataReliabilitySessionWasNotAlreadyCreated() (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if dataReliabilitySession, ok := cswp.Sessions[DataReliabilitySessionId]; ok { // check if we already have a data reliability session.
		// validate our relay number reached the data reliability relay number limit
		if dataReliabilitySession.RelayNum >= DataReliabilityRelayNumber {
			return nil, cswp.PairingEpoch, DataReliabilityAlreadySentThisEpochError
		}
		dataReliabilitySession.lock.Lock() // lock before returning.
		return dataReliabilitySession, cswp.PairingEpoch, nil
	}
	return nil, cswp.PairingEpoch, NoDataReliabilitySessionWasCreatedError
}

// get a data reliability session from an endpoint
func (cswp *ConsumerSessionsWithProvider) getDataReliabilitySingleConsumerSession(endpoint *Endpoint) (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	// we re validate the data reliability session now that we are locked.
	if dataReliabilitySession, ok := cswp.Sessions[DataReliabilitySessionId]; ok { // check if we already have a data reliability session.
		if dataReliabilitySession.RelayNum >= DataReliabilityRelayNumber {
			return nil, cswp.PairingEpoch, DataReliabilityAlreadySentThisEpochError
		}
		// we already have the dr session. so return it.
		return dataReliabilitySession, cswp.PairingEpoch, nil
	}

	singleDataReliabilitySession := &SingleConsumerSession{
		SessionId: DataReliabilitySessionId,
		Client:    cswp,
		Endpoint:  endpoint,
		RelayNum:  0,
	}
	singleDataReliabilitySession.lock.Lock() // we must lock the session so other requests wont get it.

	cswp.Sessions[singleDataReliabilitySession.SessionId] = singleDataReliabilitySession // applying the session to the pool of sessions.
	return singleDataReliabilitySession, cswp.PairingEpoch, nil
}

func (cswp *ConsumerSessionsWithProvider) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&cswp.PairingEpoch)
}

func (cswp *ConsumerSessionsWithProvider) getPublicLavaAddressAndPairingEpoch() (string, uint64) {
	cswp.Lock.Lock() // TODO: change to RLock when LavaMutex is changed
	defer cswp.Lock.Unlock()
	return cswp.PublicLavaAddress, cswp.PairingEpoch
}

// Validate the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) validateComputeUnits(cu uint64) error {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if (cswp.UsedComputeUnits + cu) > cswp.MaxComputeUnits {
		return utils.LavaFormatWarning("validateComputeUnits", MaxComputeUnitsExceededError, utils.Attribute{Key: "cu", Value: cswp.UsedComputeUnits + cu}, utils.Attribute{Key: "maxCu", Value: cswp.MaxComputeUnits})
	}
	return nil
}

// Validate and add the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) addUsedComputeUnits(cu uint64) error {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if (cswp.UsedComputeUnits + cu) > cswp.MaxComputeUnits {
		return MaxComputeUnitsExceededError
	}
	cswp.UsedComputeUnits += cu
	return nil
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
	conn, err := ConnectgRPCClient(connectCtx, addr, AllowInsecureConnectionToProviders)
	if err != nil {
		return nil, nil, err
	}
	/*defer conn.Close()*/

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

		if session.lock.TryLock() {
			if session.BlockListed { // this session cannot be used.
				numberOfBlockedSessions += 1 // increase the number of blocked sessions so we can block this provider is too many are blocklisted
				session.lock.Unlock()
				continue
			}
			// if we locked the session its available to use, otherwise someone else is already using it
			return session, cswp.PairingEpoch, nil
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
		Client:    cswp,
		Endpoint:  endpoint,
	}
	consumerSession.lock.Lock() // we must lock the session so other requests wont get it.

	cswp.Sessions[consumerSession.SessionId] = consumerSession // applying the session to the pool of sessions.
	return consumerSession, cswp.PairingEpoch, nil
}

// fetching an endpoint from a ConsumerSessionWithProvider and establishing a connection,
// can fail without an error if trying to connect once to each endpoint but none of them are active.
func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromConsumerSessionWithProvider(ctx context.Context) (connected bool, endpointPtr *Endpoint, providerAddress string, err error) {
	getConnectionFromConsumerSessionsWithProvider := func(ctx context.Context) (connected bool, endpointPtr *Endpoint, allDisabled bool) {
		cswp.Lock.Lock()
		defer cswp.Lock.Unlock()

		for idx, endpoint := range cswp.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			connectEndpoint := func(cswp *ConsumerSessionsWithProvider, ctx context.Context, endpoint *Endpoint) (connected_ bool) {
				if endpoint.Client != nil && endpoint.connection.GetState() != connectivity.Shutdown {
					return true
				}
				client, conn, err := cswp.ConnectRawClientWithTimeout(ctx, endpoint.NetworkAddress)
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatError("error connecting to provider", err, utils.Attribute{Key: "provider endpoint", Value: endpoint.NetworkAddress}, utils.Attribute{Key: "provider address", Value: cswp.PublicLavaAddress}, utils.Attribute{Key: "endpoint", Value: endpoint}, utils.Attribute{Key: "refusals", Value: endpoint.ConnectionRefusals})
					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttempts {
						endpoint.Enabled = false
						utils.LavaFormatWarning("disabling provider endpoint for the duration of current epoch.", nil, utils.Attribute{Key: "Endpoint", Value: endpoint.NetworkAddress}, utils.Attribute{Key: "address", Value: cswp.PublicLavaAddress})
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
			if endpoint.Client == nil {
				connected_ := connectEndpoint(cswp, ctx, endpoint)
				if !connected_ {
					continue
				}
			} else if endpoint.connection.GetState() == connectivity.Shutdown {
				// connection was shut down, so we need to create a new one
				endpoint.connection.Close()
				connected_ := connectEndpoint(cswp, ctx, endpoint)
				if !connected_ {
					continue
				}
			}
			cswp.Endpoints[idx] = endpoint
			return true, endpoint, false
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
	connected, endpointPtr, allDisabled = getConnectionFromConsumerSessionsWithProvider(ctx)
	if allDisabled {
		utils.LavaFormatError("purging provider after all endpoints are disabled", nil, utils.Attribute{Key: "provider endpoints", Value: cswp.Endpoints}, utils.Attribute{Key: "provider address", Value: cswp.PublicLavaAddress})
		// report provider.
		return connected, endpointPtr, cswp.PublicLavaAddress, AllProviderEndpointsDisabledError
	}

	return connected, endpointPtr, cswp.PublicLavaAddress, nil
}

// returns the expected latency to a threshold.
func (cs *SingleConsumerSession) CalculateExpectedLatency(timeoutGivenToRelay time.Duration) time.Duration {
	expectedLatency := (timeoutGivenToRelay / 2)
	return expectedLatency
}

func (cs *SingleConsumerSession) CalculateQoS(latency, expectedLatency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {
	// Add current Session QoS
	cs.QoSInfo.TotalRelays++    // increase total relays
	cs.QoSInfo.AnsweredRelays++ // increase answered relays

	if cs.QoSInfo.LastQoSReport == nil {
		cs.QoSInfo.LastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage, scaledAvailabilityScore := CalculateAvailabilityScore(&cs.QoSInfo)
	cs.QoSInfo.LastQoSReport.Availability = scaledAvailabilityScore
	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Availability) {
		utils.LavaFormatInfo("QoS Availability report", utils.Attribute{Key: "Availability", Value: cs.QoSInfo.LastQoSReport.Availability}, utils.Attribute{Key: "down percent", Value: downtimePercentage})
	}

	latencyScore := sdk.MinDec(sdk.OneDec(), sdk.NewDecFromInt(sdk.NewInt(int64(expectedLatency))).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(latency)))))

	insertSorted := func(list []sdk.Dec, value sdk.Dec) []sdk.Dec {
		index := sort.Search(len(list), func(i int) bool {
			return list[i].GTE(value)
		})
		if len(list) == index { // nil or empty slice or after last element
			return append(list, value)
		}
		list = append(list[:index+1], list[index:]...) // index < len(a)
		list[index] = value
		return list
	}
	cs.QoSInfo.LatencyScoreList = insertSorted(cs.QoSInfo.LatencyScoreList, latencyScore)
	cs.QoSInfo.LastQoSReport.Latency = cs.QoSInfo.LatencyScoreList[int(float64(len(cs.QoSInfo.LatencyScoreList))*PercentileToCalculateLatency)]

	// checking if we have enough information to calculate the sync score for the providers, if we haven't talked
	// with enough providers we don't have enough information and we will wait to have more information before setting the sync score
	shouldCalculateSyncScore := int64(numOfProviders) > int64(math.Ceil(float64(servicersToCount)*MinProvidersForSync))
	if shouldCalculateSyncScore { //
		if blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			cs.QoSInfo.SyncScoreSum++
		}
		cs.QoSInfo.TotalSyncScore++
		cs.QoSInfo.LastQoSReport.Sync = sdk.NewDec(cs.QoSInfo.SyncScoreSum).QuoInt64(cs.QoSInfo.TotalSyncScore)
		if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Sync) {
			utils.LavaFormatDebug("QoS Sync report",
				utils.Attribute{Key: "Sync", Value: cs.QoSInfo.LastQoSReport.Sync},
				utils.Attribute{Key: "block diff", Value: blockHeightDiff},
				utils.Attribute{Key: "sync score", Value: strconv.FormatInt(cs.QoSInfo.SyncScoreSum, 10) + "/" + strconv.FormatInt(cs.QoSInfo.TotalSyncScore, 10)},
				utils.Attribute{Key: "session_id", Value: blockHeightDiff},
			)
		}
	} // else, we don't increase the score at all so everyone will have the same score
}

func CalculateAvailabilityScore(qosReport *QoSReport) (downtimePercentageRet, scaledAvailabilityScoreRet sdk.Dec) {
	downtimePercentage := sdk.NewDecWithPrec(int64(qosReport.TotalRelays-qosReport.AnsweredRelays), 0).Quo(sdk.NewDecWithPrec(int64(qosReport.TotalRelays), 0))
	scaledAvailabilityScore := sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePercentage).Quo(AvailabilityPercentage))
	return downtimePercentage, scaledAvailabilityScore
}

// validate if this is a data reliability session
func (scs *SingleConsumerSession) IsDataReliabilitySession() bool {
	return scs.SessionId <= DataReliabilitySessionId
}
