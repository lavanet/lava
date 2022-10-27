package lavasession

import (
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/utils"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers map[uint64]map[string]*ProviderSessionsWithConsumer // first key is epochs, second key is a consumer address
	lock                     sync.RWMutex
	blockedEpoch             uint64 // requests from this epoch are blocked
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicWriteBlockedEpoch(epoch uint64) {
	atomic.StoreUint64(&psm.blockedEpoch, epoch)
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicReadBlockedEpoch() (epoch uint64) {
	return atomic.LoadUint64(&psm.blockedEpoch)
}

//
func (psm *ProviderSessionManager) IsValidEpoch(epoch uint64) bool {
	if epoch <= psm.atomicReadBlockedEpoch() { // TODO_RAN: atomic read.
		return false
	}
	return true
}

func (psm *ProviderSessionManager) IsActiveConsumer(epoch uint64, address string) bool {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if mapOfProviderSessionsWithConsumer, ok := psm.sessionsWithAllConsumers[epoch]; ok {
		if _, ok := mapOfProviderSessionsWithConsumer[address]; ok {
			return true
		}
	}
	return false
}

func (psm *ProviderSessionManager) GetSession(address string, id uint64, epoch uint64, relayNum uint64) (singleProviderSession *SingleProviderSession, err error) {
	if psm.IsValidEpoch(epoch) {
		utils.LavaFormatError("GetSession", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}

	if psm.IsActiveConsumer(epoch, address) {
		_, _ = psm.getSessionFromAnActiveConsumer(epoch, address)
	} else if relayNum == 0 {
		_, _ = psm.getNewSession(epoch, address)
	} else {
		utils.LavaFormatError("GetSession", NewSessionWithRelayNumError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, NewSessionWithRelayNumError
	}

	// validate later relayNum etc..

	return nil, nil
}

func (psm *ProviderSessionManager) getSessionFromAnActiveConsumer(epoch uint64, address string) (singleProviderSession *SingleProviderSession, err error) {
	return nil, nil
}

func (psm *ProviderSessionManager) getNewSession(epoch uint64, address string) (singleProviderSession *SingleProviderSession, err error) {
	return nil, nil
}

func (psm *ProviderSessionManager) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil
}

func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64) (err error) {
	return nil
}

func (psm *ProviderSessionManager) OnSessionFailure() (epoch uint64, err error) {
	return 0, nil
}

func (psm *ProviderSessionManager) OnSessionDone(proof string) (epoch uint64, err error) {
	return 0, nil
}

// Returning a new provider session manager
func GetProviderSessionManager() *ProviderSessionManager {
	return &ProviderSessionManager{}
}
