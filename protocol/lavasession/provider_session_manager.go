package lavasession

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers map[uint64]map[string]*ProviderSessionsWithConsumer // first key is epochs, second key is a consumer address
	lock                     sync.RWMutex
	blockedEpoch             uint64 // requests from this epoch are blocked
	rpcProviderEndpoint      *RPCProviderEndpoint
	stateQuery               StateQuery
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicWriteBlockedEpoch(epoch uint64) {
	atomic.StoreUint64(&psm.blockedEpoch, epoch)
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicReadBlockedEpoch() (epoch uint64) {
	return atomic.LoadUint64(&psm.blockedEpoch)
}

func (psm *ProviderSessionManager) IsValidEpoch(epoch uint64) (valid bool, thresholdEpoch uint64) {
	threshold := psm.atomicReadBlockedEpoch()
	return epoch > threshold, threshold
}

// Check if consumer exists and is not blocked, if all is valid return the ProviderSessionsWithConsumer pointer
func (psm *ProviderSessionManager) IsActiveConsumer(epoch uint64, address string) (isActive bool, providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	providerSessionWithConsumer, err = psm.getActiveConsumer(epoch, address)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			// consumer is not registered, but its not an error.
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, providerSessionWithConsumer, nil // no error
}

func (psm *ProviderSessionManager) GetSession(address string, epoch uint64, relayNum uint64, sessionId uint64) (*SingleProviderSession, error) {
	valid, _ := psm.IsValidEpoch(epoch)
	if valid { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}

	activeConsumer, providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}
	var singleProviderSession *SingleProviderSession
	if activeConsumer {
		singleProviderSession, err = psm.getSessionFromAnActiveConsumer(providerSessionWithConsumer, sessionId) // after getting session verify relayNum etc..
	} else if relayNum == 1 {
		// if no session found, we need to create and validate few things: pairing,
		// return here and call a different function.
		// in this function

		singleProviderSession, err = psm.getNewSession(epoch, address) // after getting session verify relayNum etc..
	} else {
		return nil, utils.LavaFormatError("GetSession Error, Consumer is not active and relayNum != 1",
			NewSessionWithRelayNumError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "relayNum": strconv.FormatUint(relayNum, 10)})
	}

	if err != nil {
		utils.LavaFormatError("GetSession Failure", err, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, err
	}

	// validate later relayNum etc..

	// singleProviderSession Is locked at this point (currently it isnt dont forget to add locks where needed)
	return singleProviderSession, nil
}

func (psm *ProviderSessionManager) createNewSingleProviderSession(providerSessionWithConsumer *ProviderSessionsWithConsumer, sessionId uint64) (singleProviderSession *SingleProviderSession, err error) {
	providerSessionWithConsumer.Lock.Lock()
	defer providerSessionWithConsumer.Lock.Unlock()

	// TODO: create a new single provider session

	return nil, fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) getActiveConsumer(epoch uint64, address string) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	valid, _ := psm.IsValidEpoch(epoch)
	if valid { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}
	if mapOfProviderSessionsWithConsumer, ok := psm.sessionsWithAllConsumers[epoch]; ok {
		if providerSessionWithConsumer, ok := mapOfProviderSessionsWithConsumer[address]; ok {
			if providerSessionWithConsumer.atomicReadBlockedEpoch() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "ConsumerAddress": address})
				return nil, ConsumerIsBlockListed
			}
			return providerSessionWithConsumer, nil // no error
		}
	}
	return nil, ConsumerNotRegisteredYet
}

func (psm *ProviderSessionManager) getSessionFromAnActiveConsumer(providerSessionWithConsumer *ProviderSessionsWithConsumer, sessionId uint64) (singleProviderSession *SingleProviderSession, err error) {
	session, err := providerSessionWithConsumer.GetExistingSession(sessionId)
	if err == nil {
		return session, nil
	} else if SessionDoesNotExist.Is(err) {
		utils.LavaFormatFatal("GetExistingSession Unexpected Error", err, nil)
	}
	// if we don't have a session we need to create a new one.
	return psm.createNewSingleProviderSession(providerSessionWithConsumer, sessionId)
}

func (psm *ProviderSessionManager) getNewSession(epoch uint64, address string) (singleProviderSession *SingleProviderSession, err error) {
	return
}

func (psm *ProviderSessionManager) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil
}

func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64) (*SingleProviderSession, error) {
	return nil, fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) OnSessionFailure(singleProviderSession *SingleProviderSession) (err error) {
	return nil
}

func (psm *ProviderSessionManager) OnSessionDone(proof string) (epoch uint64, err error) {
	return 0, nil
}

func (psm *ProviderSessionManager) RPCProviderEndpoint() *RPCProviderEndpoint {
	return psm.rpcProviderEndpoint
}

func (psm *ProviderSessionManager) UpdateEpoch(epoch uint64) {
	// update the epoch to limit consumer usage
}
func (psm *ProviderSessionManager) ProcessUnsubscribeEthereum(subscriptionID string, consumerAddress sdk.AccAddress) error {
	return fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) ProcessUnsubscribeTendermint(apiName string, subscriptionID string, consumerAddress sdk.AccAddress) error {
	return fmt.Errorf("not implemented")
}

// Returning a new provider session manager
func NewProviderSessionManager(rpcProviderEndpoint *RPCProviderEndpoint, stateQuery StateQuery) *ProviderSessionManager {
	return &ProviderSessionManager{rpcProviderEndpoint: rpcProviderEndpoint, stateQuery: stateQuery}
}
