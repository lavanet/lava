package lavasession

import (
	"context"
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
	Addr               string
	Enabled            bool
	Client             *pairingtypes.RelayerClient
	ConnectionRefusals uint64
}

type ConsumerSessionsWithProvider struct {
	Lock             utils.LavaMutex
	Acc              string //public lava address
	Endpoints        []*Endpoint
	Sessions         map[int64]*ConsumerSession
	MaxComputeUnits  uint64
	UsedComputeUnits uint64
	ReliabilitySent  bool
	PairingEpoch     uint64
}

func (cswp *ConsumerSessionsWithProvider) fetchEndpointConnectionFromClientWrapper() (connected bool, endpointPtr *Endpoint) {
	//assumes s.pairingMu is Rlocked here
	getConnectionFromWrap := func(s *Sentry, ctx context.Context, index int) (connected bool, endpointPtr *Endpoint, allDisabled bool) {
		wrap.SessionsLock.Lock()
		defer wrap.SessionsLock.Unlock()

		for idx, endpoint := range wrap.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			if endpoint.Client == nil {
				connectCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
				conn, err := s.connectRawClient(connectCtx, endpoint.Addr)
				cancel()
				if err != nil {
					endpoint.ConnectionRefusals++
					utils.LavaFormatError("error connecting to provider", err, &map[string]string{"provider endpoint": endpoint.Addr, "provider address": wrap.Acc, "endpoint": fmt.Sprintf("%+v", endpoint)})
					if endpoint.ConnectionRefusals >= MaxConsecutiveConnectionAttemts {
						endpoint.Enabled = false
						utils.LavaFormatWarning("disabling provider endpoint", nil, &map[string]string{"Endpoint": endpoint.Addr, "address": wrap.Acc, "currentEpoch": strconv.FormatInt(s.GetBlockHeight(), 10)})
					}
					continue
				}

				endpoint.ConnectionRefusals = 0
				endpoint.Client = conn
			}
			wrap.Endpoints[idx] = endpoint
			return true, endpoint, false
		}

		// checking disabled endpoints, as we can disable an endpoint mid run of the previous loop, we should re test the current endpoint state
		// before verifing all are Disabled.
		allDisabled = true
		for _, endpoint := range wrap.Endpoints {
			if !endpoint.Enabled {
				continue
			}
			// even one endpoint is enough for us to not purge.
			allDisabled = false
		}
		return false, nil, allDisabled
	}
	var allDisabled bool
	connected, endpointPtr, allDisabled = getConnectionFromWrap(s, ctx, index)
	//we dont purge if we tried connecting and failed, only if we already disabled all endpoints
	if allDisabled {
		utils.LavaFormatError("purging provider after all endpoints are disabled", nil, &map[string]string{"provider endpoints": fmt.Sprintf("%v", wrap.Endpoints), "provider address": wrap.Acc})
		// we release read lock here, we assume pairing can change in movePairingEntryToPurge and it needs rw lock
		// we resume read lock right after so we can continue reading
		s.rUnlockAndMovePairingEntryToPurgeReturnRLocked(wrap, index, true)
	}

	return connected, endpointPtr
}
