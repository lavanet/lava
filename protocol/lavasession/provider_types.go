package lavasession

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type voteData struct {
	RelayDataHash []byte
	Nonce         int64
	CommitHash    []byte
}

type ProviderSessionsEpochData struct {
	UsedComputeUnits uint64
	MaxComputeUnits  uint64
	DataReliability  *pairingtypes.VRFData
	VrfPk            utils.VrfPubKey
}

type RPCProviderEndpoint struct {
	NetworkAddress string   `yaml:"network-address,omitempty" json:"network-address,omitempty" mapstructure:"network-address,omitempty"` // IP:PORT
	ChainID        string   `yaml:"chain-id,omitempty" json:"chain-id,omitempty" mapstructure:"chain-id"`                                // spec chain identifier
	ApiInterface   string   `yaml:"api-interface,omitempty" json:"api-interface,omitempty" mapstructure:"api-interface"`
	Geolocation    uint64   `yaml:"geolocation,omitempty" json:"geolocation,omitempty" mapstructure:"geolocation"`
	NodeUrl        []string `yaml:"node-url,omitempty" json:"node-url,omitempty" mapstructure:"node-url"`
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
)

// holds all of the data for a consumer for a certain epoch
type ProviderSessionsWithConsumer struct {
	Sessions      map[uint64]*SingleProviderSession
	isBlockListed uint32
	consumer      string
	epochData     *ProviderSessionsEpochData
	Lock          sync.RWMutex
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

// reads cs.BlockedEpoch atomically, notBlockListedConsumer = 0, blockListedConsumer    = 1
func (pswc *ProviderSessionsWithConsumer) atomicWriteBlockedEpoch(blockStatus uint32) { // rename to blocked consumer not blocked epoch
	atomic.StoreUint32(&pswc.isBlockListed, blockStatus)
}

// reads cs.BlockedEpoch atomically
func (pswc *ProviderSessionsWithConsumer) atomicReadBlockedEpoch() (blockStatus uint32) {
	return atomic.LoadUint32(&pswc.isBlockListed)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadMaxComputeUnits() (maxComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.MaxComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicReadUsedComputeUnits() (usedComputeUnits uint64) {
	return atomic.LoadUint64(&pswc.epochData.UsedComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicWriteMaxComputeUnits(maxComputeUnits uint64) {
	atomic.StoreUint64(&pswc.epochData.MaxComputeUnits, maxComputeUnits)
}

func (pswc *ProviderSessionsWithConsumer) atomicCompareAndWriteUsedComputeUnits(newUsed uint64, knownUsed uint64) bool {
	return atomic.CompareAndSwapUint64(&pswc.epochData.UsedComputeUnits, knownUsed, newUsed)
}

func (pswc *ProviderSessionsWithConsumer) createNewSingleProviderSession(sessionId uint64, epoch uint64) (session *SingleProviderSession, err error) {
	session = &SingleProviderSession{
		userSessionsParent: pswc,
		SessionID:          sessionId,
		PairingEpoch:       epoch,
	}
	session.lock.Lock()
	return session, nil
}

func (pswc *ProviderSessionsWithConsumer) GetExistingSession(sessionId uint64) (session *SingleProviderSession, err error) {
	pswc.Lock.RLock()
	defer pswc.Lock.RUnlock()
	if session, ok := pswc.Sessions[sessionId]; ok {
		session.lock.Lock()
		return session, nil
	}
	return nil, SessionDoesNotExist
}

func (sps *SingleProviderSession) GetPairingEpoch() uint64 {
	return atomic.LoadUint64(&sps.PairingEpoch)
}

func (sps *SingleProviderSession) SetPairingEpoch(epoch uint64) {
	atomic.StoreUint64(&sps.PairingEpoch, epoch)
}

// Verify the SingleProviderSession is locked when getting to this function, if its not locked throw an error
func (sps *SingleProviderSession) verifyLock() error {
	if sps.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer sps.lock.Unlock()
		return LockMisUseDetectedError
	}
	return nil
}

func (sps *SingleProviderSession) PrepareSessionForUsage(currentCU uint64, relayRequestTotalCU uint64, relayNumber uint64) error {
	err := sps.verifyLock() // sps is locked
	if err != nil {
		return utils.LavaFormatError("sps.verifyLock() failed in PrepareSessionForUsage", err, nil)
	}

	if sps.RelayNum+1 != relayNumber {
		sps.lock.Unlock() // unlock on error
		return utils.LavaFormatError("Maximum cu exceeded PrepareSessionForUsage", MaximumCULimitReachedByConsumer, &map[string]string{
			"relayNumber":  strconv.FormatUint(relayNumber, 10),
			"sps.RelayNum": strconv.FormatUint(sps.RelayNum+1, 10),
		})
	}

	maxCu := sps.userSessionsParent.atomicReadMaxComputeUnits()
	if relayRequestTotalCU < sps.CuSum+currentCU {
		sps.lock.Unlock() // unlock on error
		return utils.LavaFormatError("CU mismatch PrepareSessionForUsage, Provider and consumer disagree on CuSum", ProviderConsumerCuMisMatch, &map[string]string{
			"relayRequestTotalCU": strconv.FormatUint(relayRequestTotalCU, 10),
			"sps.CuSum":           strconv.FormatUint(sps.CuSum, 10),
			"currentCU":           strconv.FormatUint(currentCU, 10),
		})
	}

	// this must happen first, as we also validate and add the used cu to parent here
	err = sps.validateAndAddUsedCU(currentCU, maxCu)
	if err != nil {
		sps.lock.Unlock() // unlock on error
		return err
	}
	// finished validating, can add all info.
	sps.LatestRelayCu = currentCU   // 1. update latest
	sps.CuSum = relayRequestTotalCU // 2. update CuSum, if consumer wants to pay more, let it
	sps.RelayNum = sps.RelayNum + 1

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

type StateQuery interface {
	QueryVerifyPairing(ctx context.Context, consumer string, blockHeight uint64)
}
