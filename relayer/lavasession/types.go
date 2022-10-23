package lavasession

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc"
)

type QoSInfo struct {
	LastQoSReport      *pairingtypes.QualityOfServiceReport
	LatencyScoreList   []sdk.Dec
	SyncScoreSum       int64
	TotalSyncScore     int64
	TotalRelays        uint64
	AnsweredRelays     uint64
	ConsecutiveTimeOut uint64
}

type ConsumerSession struct {
	cuSum         uint64
	latestRelayCu uint64 // set by GetSession cuNeededForSession
	qoSInfo       QoSInfo
	sessionId     int64
	client        *ConsumerSessionsWithProvider
	lock          utils.LavaMutex
	relayNum      uint64
	latestBlock   int64
	endpoint      *Endpoint
	blocklisted   bool // if session lost sync we blacklist it.
}

func (cs *ConsumerSession) GetCuSum() {

}

type Endpoint struct {
	Addr               string // change at the end to NetworkAddress
	Enabled            bool
	Client             *pairingtypes.RelayerClient
	ConnectionRefusals uint64
}

type ConsumerSessionsWithProvider struct {
	Lock             utils.LavaMutex
	Acc              string //public lava address // change at the end to PublicLavaAddress
	Endpoints        []*Endpoint
	Sessions         map[int64]*ConsumerSession
	MaxComputeUnits  uint64
	UsedComputeUnits uint64
	ReliabilitySent  bool
	PairingEpoch     uint64
}

func (cswp *ConsumerSessionsWithProvider) getPublicLavaAddressAndPairingEpoch() (string, uint64) {
	cswp.Lock.Lock() // TODO: change to RLock when LavaMutex is chagned
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
	if (cswp.UsedComputeUnits - cu) < 0 {
		return NegativeComputeUnitsAmountError
	}
	cswp.UsedComputeUnits -= cu
	return nil
}

func (cswp *ConsumerSessionsWithProvider) connectRawClient(ctx context.Context, addr string) (*pairingtypes.RelayerClient, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(connectCtx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	/*defer conn.Close()*/

	c := pairingtypes.NewRelayerClient(conn)
	return &c, nil
}

func (cswp *ConsumerSessionsWithProvider) getConsumerSessionInstanceFromEndpoint(endpoint *Endpoint) (*ConsumerSession, error) {
	// TODO_RAN: validate that the endpoint even belongs to the ConsumerSessionsWithProvider and is enabled.

	cswp.Lock.Lock()
	defer cswp.Lock.Unlock()
	//try to lock an existing session, if can't create a new one
	for _, session := range cswp.Sessions {
		if session.endpoint != endpoint {
			//skip sessions that don't belong to the active connection
			continue
		}
		if session.lock.TryLock() {
			// if we locked the session its available to use, otherwise someone else is already using it
			return session, nil
		}
	}
	// No Sessions available, create a new session or return an error upon maximum sessions allowed
	if len(cswp.Sessions) > MaxSessionsAllowedPerProvider {
		return nil, MaximumNumberOfSessionsExceededError
	}

	randomSessId := int64(0)
	for randomSessId == 0 { //we don't allow 0
		randomSessId = rand.Int63()
	}

	consumerSession := &ConsumerSession{
		sessionId: randomSessId,
		client:    cswp,
		endpoint:  endpoint,
	}
	consumerSession.lock.Lock() // we must lock the session so other requests wont get it.

	cswp.Sessions[consumerSession.sessionId] = consumerSession // applying the session to the pool of sessions.
	return consumerSession, nil
}

// fetching an enpoint from a ConsumerSessionWithProvider and establishing a connection,
// can fail without an error if trying to connect once to each endpoint but none of them are active.
func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromConsumerSessionWithProvider(ctx context.Context, sessionEpoch uint64) (connected bool, endpointPtr *Endpoint, err error) {
	getConnectionFromcswp := func(ctx context.Context) (connected bool, endpointPtr *Endpoint, allDisabled bool) {
		cswp.Lock.Lock()
		defer cswp.Lock.Unlock()

		for idx, endpoint := range cswp.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			if endpoint.Client == nil {
				connectCtx, cancel := context.WithTimeout(ctx, TimeoutForEstablishingAConnectionInMS)
				conn, err := cswp.connectRawClient(connectCtx, endpoint.Addr)
				cancel()
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatError("error connecting to provider", err, &map[string]string{"provider endpoint": endpoint.Addr, "provider address": cswp.Acc, "endpoint": fmt.Sprintf("%+v", endpoint)})
					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttemts {
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
		// before verifing all are Disabled.
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
	connected, endpointPtr, allDisabled = getConnectionFromcswp(ctx)
	if allDisabled {
		utils.LavaFormatError("purging provider after all endpoints are disabled", nil, &map[string]string{"provider endpoints": fmt.Sprintf("%v", cswp.Endpoints), "provider address": cswp.Acc})
		// report provider.
		return connected, endpointPtr, AllProviderEndpointsDisabledError
	}

	return connected, endpointPtr, nil
}
