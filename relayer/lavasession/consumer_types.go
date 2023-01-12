package lavasession

import (
	"context"
	"fmt"
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
	"google.golang.org/grpc/credentials/insecure"
)

type ignoredProviders struct {
	providers    map[string]struct{}
	currentEpoch uint64
}

type qoSInfo struct {
	LastQoSReport    *pairingtypes.QualityOfServiceReport
	LatencyScoreList []sdk.Dec
	SyncScoreSum     int64
	TotalSyncScore   int64
	TotalRelays      uint64
	AnsweredRelays   uint64
}

type SingleConsumerSession struct {
	CuSum                       uint64
	LatestRelayCu               uint64 // set by GetSession cuNeededForSession
	QoSInfo                     qoSInfo
	SessionId                   int64
	Client                      *ConsumerSessionsWithProvider
	lock                        utils.LavaMutex
	RelayNum                    uint64
	LatestBlock                 int64
	Endpoint                    *Endpoint
	BlockListed                 bool   // if session lost sync we blacklist it.
	ConsecutiveNumberOfFailures uint64 // number of times this session has failed
}

type Endpoint struct {
	Addr               string // change at the end to NetworkAddress
	Enabled            bool
	Client             *pairingtypes.RelayerClient
	ConnectionRefusals uint64
}

type ConsumerSessionsWithProvider struct {
	Lock             utils.LavaMutex
	Acc              string // public lava address // change at the end to PublicLavaAddress
	Endpoints        []*Endpoint
	Sessions         map[int64]*SingleConsumerSession
	MaxComputeUnits  uint64
	UsedComputeUnits uint64
	ReliabilitySent  bool
	PairingEpoch     uint64
}

// verify data reliability session exists or not
func (cswp *ConsumerSessionsWithProvider) verifyDataReliabilitySessionWasNotAlreadyCreated() (err error) {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if _, ok := cswp.Sessions[DataReliabilitySessionId]; ok { // check if we already have a data reliability session.
		return DataReliabilityAlreadySentThisEpochError
	}
	return nil
}

// get a data reliability session from an endpoint
func (cswp *ConsumerSessionsWithProvider) getDataReliabilitySingleConsumerSession(endpoint *Endpoint) (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	// we re validate the data reliability session now that we are locked.
	if _, ok := cswp.Sessions[DataReliabilitySessionId]; ok { // check if we already have a data reliability session.
		return nil, cswp.PairingEpoch, DataReliabilityAlreadySentThisEpochError
	}

	singleDataReliabilitySession := &SingleConsumerSession{
		SessionId: DataReliabilitySessionId,
		Client:    cswp,
		Endpoint:  endpoint,
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
	return cswp.Acc, cswp.PairingEpoch
}

// Validate the compute units for this provider
func (cswp *ConsumerSessionsWithProvider) validateComputeUnits(cu uint64) error {
	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	if (cswp.UsedComputeUnits + cu) > cswp.MaxComputeUnits {
		return MaxComputeUnitsExceededError
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

func (cswp *ConsumerSessionsWithProvider) connectRawClientWithTimeout(ctx context.Context, addr string) (*pairingtypes.RelayerClient, error) {
	connectCtx, cancel := context.WithTimeout(ctx, TimeoutForEstablishingAConnection)
	defer cancel()

	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerClient(conn)
	return &c, nil
}

func (cswp *ConsumerSessionsWithProvider) getConsumerSessionInstanceFromEndpoint(endpoint *Endpoint, maximumBlockedSessionsMultiplier uint64) (singleConsumerSession *SingleConsumerSession, pairingEpoch uint64, err error) {
	// TODO: validate that the endpoint even belongs to the ConsumerSessionsWithProvider and is enabled.

	maximumBlockedSessionsAllowed := MaxAllowedBlockListedSessionPerProvider * (maximumBlockedSessionsMultiplier + 1) // +1 as we start from 0
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
func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromConsumerSessionWithProvider(ctx context.Context, sessionEpoch uint64) (connected bool, endpointPtr *Endpoint, err error) {
	getConnectionFromConsumerSessionsWithProvider := func(ctx context.Context) (connected bool, endpointPtr *Endpoint, allDisabled bool) {
		cswp.Lock.Lock()
		defer cswp.Lock.Unlock()

		for idx, endpoint := range cswp.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			if endpoint.Client == nil {
				conn, err := cswp.connectRawClientWithTimeout(ctx, endpoint.Addr)
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatError("error connecting to provider", err, &map[string]string{"provider endpoint": endpoint.Addr, "provider address": cswp.Acc, "endpoint": fmt.Sprintf("%+v", endpoint)})
					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttempts {
						endpoint.Enabled = false
						utils.LavaFormatWarning("disabling provider endpoint for the duration of current epoch.", nil, &map[string]string{"Endpoint": endpoint.Addr, "address": cswp.Acc, "currentEpoch": strconv.FormatUint(sessionEpoch, 10)})
					}
					continue
				}
				endpoint.ConnectionRefusals = 0
				endpoint.Client = conn
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
		utils.LavaFormatError("purging provider after all endpoints are disabled", nil, &map[string]string{"provider endpoints": fmt.Sprintf("%v", cswp.Endpoints), "provider address": cswp.Acc})
		// report provider.
		return connected, endpointPtr, AllProviderEndpointsDisabledError
	}

	return connected, endpointPtr, nil
}

func (cs *SingleConsumerSession) CalculateQoS(cu uint64, latency time.Duration, blockHeightDiff int64, numOfProviders int, servicersToCount int64) {
	// Add current Session QoS
	cs.QoSInfo.TotalRelays++    // increase total relays
	cs.QoSInfo.AnsweredRelays++ // increase answered relays

	if cs.QoSInfo.LastQoSReport == nil {
		cs.QoSInfo.LastQoSReport = &pairingtypes.QualityOfServiceReport{}
	}

	downtimePercentage := sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays-cs.QoSInfo.AnsweredRelays), 0).Quo(sdk.NewDecWithPrec(int64(cs.QoSInfo.TotalRelays), 0))
	cs.QoSInfo.LastQoSReport.Availability = sdk.MaxDec(sdk.ZeroDec(), AvailabilityPercentage.Sub(downtimePercentage).Quo(AvailabilityPercentage))
	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Availability) {
		utils.LavaFormatInfo("QoS Availability report", &map[string]string{"Availability": cs.QoSInfo.LastQoSReport.Availability.String(), "down percent": downtimePercentage.String()})
	}

	latencyThreshold := LatencyThresholdStatic + time.Duration(cu)*LatencyThresholdSlope
	latencyScore := sdk.MinDec(sdk.OneDec(), sdk.NewDecFromInt(sdk.NewInt(int64(latencyThreshold))).Quo(sdk.NewDecFromInt(sdk.NewInt(int64(latency)))))

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

	if int64(numOfProviders) > int64(math.Ceil(float64(servicersToCount)*MinProvidersForSync)) { //
		if blockHeightDiff <= 0 { // if the diff is bigger than 0 than the block is too old (blockHeightDiff = expected - allowedLag - blockHeight) and we don't give him the score
			cs.QoSInfo.SyncScoreSum++
		}
	} else {
		cs.QoSInfo.SyncScoreSum++
	}
	cs.QoSInfo.TotalSyncScore++

	cs.QoSInfo.LastQoSReport.Sync = sdk.NewDec(cs.QoSInfo.SyncScoreSum).QuoInt64(cs.QoSInfo.TotalSyncScore)

	if sdk.OneDec().GT(cs.QoSInfo.LastQoSReport.Sync) {
		utils.LavaFormatInfo("QoS Sync report",
			&map[string]string{
				"Sync":       cs.QoSInfo.LastQoSReport.Sync.String(),
				"block diff": strconv.FormatInt(blockHeightDiff, 10),
				"sync score": strconv.FormatInt(cs.QoSInfo.SyncScoreSum, 10) + "/" + strconv.FormatInt(cs.QoSInfo.TotalSyncScore, 10),
			})
	}
}
