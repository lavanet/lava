package lavasession

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
)

type voteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
}

type ProviderSessionsEpochData struct {
	UsedComputeUnits uint64
	MaxComputeUnits  uint64
}

type RPCProviderEndpoint struct {
	NetworkAddress string   `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"` // HOST:PORT
	ChainID        string   `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                                // spec chain identifier
	ApiInterface   string   `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64   `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrl        []string `yaml:"node-url,omitempty" json:"node-url,omitempty" mapstructure:"node-url"`
}

func (endpoint *RPCProviderEndpoint) String() (retStr string) {
	return endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress + " Node: " + strings.Join(endpoint.NodeUrl, ", ") + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10)
}

func (endpoint *RPCProviderEndpoint) Validate() error {
	if len(endpoint.NodeUrl) == 0 {
		return utils.LavaFormatError("Empty URL list for endpoint", nil, &map[string]string{"endpoint": endpoint.String()})
	}
	for _, url := range endpoint.NodeUrl {
		err := common.ValidateEndpoint(url, endpoint.ApiInterface)
		if err != nil {
			return err
		}
	}
	return nil
}

type RPCSubscription struct {
	Id                   string
	Sub                  *rpcclient.ClientSubscription
	SubscribeRepliesChan chan interface{}
}

func (rpcpe *RPCProviderEndpoint) Key() string {
	return rpcpe.ChainID + rpcpe.ApiInterface
}

const (
	notBlockListedConsumer = 0
	blockListedConsumer    = 1
	notDataReliabilityPSWC = 0
	isDataReliabilityPSWC  = 1
)

// holds all of the data for a consumer for a certain epoch
type ProviderSessionsWithConsumer struct {
	Sessions          map[uint64]*SingleProviderSession
	isBlockListed     uint32
	consumerAddr      string
	epochData         *ProviderSessionsEpochData
	Lock              sync.RWMutex
	isDataReliability uint32 // 0 is false, 1 is true. set to uint so we can atomically read
}

type SingleProviderSession struct {
	userSessionsParent *ProviderSessionsWithConsumer
	CuSum              uint64
	LatestRelayCu      uint64
	SessionID          uint64
	lock               sync.RWMutex
	RelayNum           uint64
	PairingEpoch       uint64
}

func NewProviderSessionsWithConsumer(consumerAddr string, epochData *ProviderSessionsEpochData, isDataReliability uint32) *ProviderSessionsWithConsumer {
	pswc := &ProviderSessionsWithConsumer{
		Sessions:          map[uint64]*SingleProviderSession{},
		isBlockListed:     0,
		consumerAddr:      consumerAddr,
		epochData:         epochData,
		isDataReliability: isDataReliability,
	}
	return pswc
}

// reads the isDataReliability data atomically
func (pswc *ProviderSessionsWithConsumer) atomicReadIsDataReliability() uint32 { // rename to blocked consumer not blocked epoch
	return atomic.LoadUint32(&pswc.isDataReliability)
}

// reads cs.BlockedEpoch atomically, notBlockListedConsumer = 0, blockListedConsumer = 1
func (pswc *ProviderSessionsWithConsumer) atomicWriteConsumerBlocked(blockStatus uint32) { // rename to blocked consumer not blocked epoch
	atomic.StoreUint32(&pswc.isBlockListed, blockStatus)
}

// reads cs.BlockedEpoch atomically to determine if the consumer is blocked notBlockListedConsumer = 0, blockListedConsumer = 1
func (pswc *ProviderSessionsWithConsumer) atomicReadConsumerBlocked() (blockStatus uint32) {
	return atomic.LoadUint32(&pswc.isBlockListed)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadMaxComputeUnits() (maxComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.MaxComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadUsedComputeUnits() (usedComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.UsedComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicWriteUsedComputeUnits(cu uint64) {
	atomic.StoreUint64(&pswc.epochData.UsedComputeUnits, cu)
}

func (pswc *ProviderSessionsWithConsumer) atomicWriteMaxComputeUnits(maxComputeUnits uint64) {
	atomic.StoreUint64(&pswc.epochData.MaxComputeUnits, maxComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicCompareAndWriteUsedComputeUnits(newUsed uint64, knownUsed uint64) bool {
	return atomic.CompareAndSwapUint64(&pswc.epochData.UsedComputeUnits, knownUsed, newUsed)
}

// create a new session with a consumer, and store it inside it's providerSessions parent
func (pswc *ProviderSessionsWithConsumer) createNewSingleProviderSession(sessionId uint64, epoch uint64) (session *SingleProviderSession, err error) {
	utils.LavaFormatDebug("Provider creating new sessionID", &map[string]string{"SessionID": strconv.FormatUint(sessionId, 10), "epoch": strconv.FormatUint(epoch, 10)})
	session = &SingleProviderSession{
		userSessionsParent: pswc,
		SessionID:          sessionId,
		PairingEpoch:       epoch,
	}
	pswc.Lock.Lock()
	defer pswc.Lock.Unlock()
	// this is a double lock and risky but we just created session and nobody has reference to it yet
	session.lock.Lock()
	pswc.Sessions[sessionId] = session
	// session is still locked when we return it
	return session, nil
}

// this function returns the session locked to be used
func (pswc *ProviderSessionsWithConsumer) GetExistingSession(sessionId uint64) (session *SingleProviderSession, err error) {
	pswc.Lock.RLock()
	defer pswc.Lock.RUnlock()
	if session, ok := pswc.Sessions[sessionId]; ok {
		locked := session.lock.TryLock()
		if !locked {
			return nil, utils.LavaFormatError("GetExistingSession failed to lock when getting session", LockMisUseDetectedError, nil)
		}
		return session, nil
	}
	return nil, SessionDoesNotExist
}

// this function verifies the provider can create a data reliability session and returns one if valid
func (pswc *ProviderSessionsWithConsumer) getDataReliabilitySingleSession(sessionId uint64, epoch uint64) (session *SingleProviderSession, err error) {
	utils.LavaFormatDebug("Provider creating new DataReliabilitySingleSession", &map[string]string{"SessionID": strconv.FormatUint(sessionId, 10), "epoch": strconv.FormatUint(epoch, 10)})
	session, foundDataReliabilitySession := pswc.Sessions[sessionId]
	if foundDataReliabilitySession {
		// if session exists, relay number should be 0 as it might had an error
		// locking the session and returning for validation
		session.lock.Lock()
		return session, nil
	}

	// otherwise return a new session and add it to the sessions list
	session = &SingleProviderSession{
		userSessionsParent: pswc,
		SessionID:          sessionId,
		PairingEpoch:       epoch,
	}
	pswc.Lock.Lock()
	defer pswc.Lock.Unlock()
	// this is a double lock and risky but we just created session and nobody has reference to it yet
	session.lock.Lock()
	pswc.Sessions[sessionId] = session

	// session is still locked when we return it
	return session, nil
}

func (sps *SingleProviderSession) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&sps.PairingEpoch)
}

func (sps *SingleProviderSession) SetPairingEpoch(epoch uint64) {
	atomic.StoreUint64(&sps.PairingEpoch, epoch)
}

// Verify the SingleProviderSession is locked when getting to this function, if its not locked throw an error
func (sps *SingleProviderSession) VerifyLock() error {
	if sps.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer sps.lock.Unlock()
		return LockMisUseDetectedError
	}
	return nil
}

// In case the user session is a data reliability we just need to verify that the cusum is the amount agreed between the consumer and the provider
func (sps *SingleProviderSession) PrepareDataReliabilitySessionForUsage(relayRequestTotalCU uint64) error {
	if relayRequestTotalCU != DataReliabilityCuSum {
		return utils.LavaFormatError("PrepareDataReliabilitySessionForUsage", DataReliabilityCuSumMisMatchError, &map[string]string{"relayRequestTotalCU": strconv.FormatUint(relayRequestTotalCU, 10)})
	}
	sps.LatestRelayCu = DataReliabilityCuSum // 1. update latest
	sps.CuSum = relayRequestTotalCU          // 2. update CuSum, if consumer wants to pay more, let it
	sps.RelayNum = sps.RelayNum + 1          // 3. update RelayNum, we already verified relayNum is valid in GetDataReliabilitySession.
	return nil
}

func (sps *SingleProviderSession) PrepareSessionForUsage(cuFromSpec uint64, relayRequestTotalCU uint64) error {
	err := sps.VerifyLock() // sps is locked
	if err != nil {
		return utils.LavaFormatError("sps.verifyLock() failed in PrepareSessionForUsage", err, nil)
	}

	// checking if this user session is a data reliability user session.
	if sps.userSessionsParent.atomicReadIsDataReliability() == isDataReliabilityPSWC {
		return sps.PrepareDataReliabilitySessionForUsage(relayRequestTotalCU)
	}

	maxCu := sps.userSessionsParent.atomicReadMaxComputeUnits()
	if relayRequestTotalCU < sps.CuSum+cuFromSpec {
		sps.lock.Unlock() // unlock on error
		return utils.LavaFormatError("CU mismatch PrepareSessionForUsage, Provider and consumer disagree on CuSum", ProviderConsumerCuMisMatch, &map[string]string{
			"relayRequestTotalCU": strconv.FormatUint(relayRequestTotalCU, 10),
			"sps.CuSum":           strconv.FormatUint(sps.CuSum, 10),
			"currentCU":           strconv.FormatUint(cuFromSpec, 10),
		})
	}

	// this must happen first, as we also validate and add the used cu to parent here
	err = sps.validateAndAddUsedCU(cuFromSpec, maxCu)
	if err != nil {
		sps.lock.Unlock() // unlock on error
		return err
	}
	// finished validating, can add all info.
	sps.LatestRelayCu = cuFromSpec  // 1. update latest
	sps.CuSum = relayRequestTotalCU // 2. update CuSum, if consumer wants to pay more, let it
	sps.RelayNum = sps.RelayNum + 1 // 3. update RelayNum, we already verified relayNum is valid in GetSession.
	return nil
}

func (sps *SingleProviderSession) validateAndAddUsedCU(currentCU uint64, maxCu uint64) error {
	for {
		usedCu := sps.userSessionsParent.atomicReadUsedComputeUnits() // check used cu now
		if usedCu+currentCU > maxCu {
			return utils.LavaFormatError("Maximum cu exceeded PrepareSessionForUsage", MaximumCULimitReachedByConsumer, &map[string]string{
				"usedCu":    strconv.FormatUint(usedCu, 10),
				"currentCU": strconv.FormatUint(currentCU, 10),
				"maxCu":     strconv.FormatUint(maxCu, 10),
			})
		}
		// compare usedCu + current cu vs usedCu, if swap succeeds, return otherwise try again
		// this can happen when multiple sessions are adding their cu at the same time.
		// comparing and adding is protecting against race conditions as the parent is not locked.
		if sps.userSessionsParent.atomicCompareAndWriteUsedComputeUnits(usedCu+currentCU, usedCu) {
			return nil
		}
	}
}

func (sps *SingleProviderSession) validateAndSubUsedCU(currentCU uint64) error {
	for {
		usedCu := sps.userSessionsParent.atomicReadUsedComputeUnits()                               // check used cu now
		if sps.userSessionsParent.atomicCompareAndWriteUsedComputeUnits(usedCu-currentCU, usedCu) { // decrease the amount of used cu from the known value
			return nil
		}
	}
}

func (sps *SingleProviderSession) onDataReliabilitySessionFailure() error {
	sps.CuSum = sps.CuSum - sps.LatestRelayCu
	sps.RelayNum = sps.RelayNum - 1
	sps.LatestRelayCu = 0
	return nil
}

func (sps *SingleProviderSession) onSessionFailure() error {
	err := sps.VerifyLock() // sps is locked
	if err != nil {
		return utils.LavaFormatError("sps.verifyLock() failed in onSessionFailure", err, nil)
	}
	defer sps.lock.Unlock()

	// handle data reliability session failure
	if sps.userSessionsParent.atomicReadIsDataReliability() == isDataReliabilityPSWC {
		return sps.onDataReliabilitySessionFailure()
	}

	sps.CuSum = sps.CuSum - sps.LatestRelayCu
	sps.RelayNum = sps.RelayNum - 1
	sps.validateAndSubUsedCU(sps.LatestRelayCu)
	sps.LatestRelayCu = 0
	return nil
}

func (sps *SingleProviderSession) onSessionDone() error {
	err := sps.VerifyLock() // sps is locked
	if err != nil {
		return utils.LavaFormatError("sps.verifyLock() failed in onSessionDone", err, nil)
	}
	sps.LatestRelayCu = 0 // reset the cu, we can also verify its 0 when loading.
	sps.lock.Unlock()
	return nil
}
