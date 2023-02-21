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
	sessionsWithAllConsumers      map[uint64]map[string]*ProviderSessionsWithConsumer // first key is epochs, second key is a consumer address
	lock                          sync.RWMutex
	blockedEpochHeight            uint64 // requests from this epoch are blocked
	rpcProviderEndpoint           *RPCProviderEndpoint
	blockDistanceForEpochValidity uint64 // sessionsWithAllConsumers with epochs older than ((latest epoch) - numberOfBlocksKeptInMemory) are deleted.
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicWriteBlockedEpoch(epoch uint64) {
	atomic.StoreUint64(&psm.blockedEpochHeight, epoch)
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicReadBlockedEpoch() (epoch uint64) {
	return atomic.LoadUint64(&psm.blockedEpochHeight)
}

func (psm *ProviderSessionManager) IsValidEpoch(epoch uint64) (valid bool, blockedEpochHeight uint64) {
	blockedEpochHeight = psm.atomicReadBlockedEpoch()
	return epoch > blockedEpochHeight, blockedEpochHeight
}

// Check if consumer exists and is not blocked, if all is valid return the ProviderSessionsWithConsumer pointer
func (psm *ProviderSessionManager) IsActiveConsumer(epoch uint64, address string) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	providerSessionWithConsumer, err = psm.getActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}
	return providerSessionWithConsumer, nil // no error
}

func (psm *ProviderSessionManager) getSingleSessionFromProviderSessionWithConsumer(providerSessionWithConsumer *ProviderSessionsWithConsumer, sessionId uint64, epoch uint64, relayNumber uint64) (*SingleProviderSession, error) {
	if providerSessionWithConsumer.atomicReadConsumerBlocked() != notBlockListedConsumer {
		return nil, utils.LavaFormatError("This consumer address is blocked.", nil, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "consumer": providerSessionWithConsumer.consumer})
	}
	// before getting any sessions.
	singleProviderSession, err := psm.getSessionFromAnActiveConsumer(providerSessionWithConsumer, sessionId, epoch) // after getting session verify relayNum etc..
	if err != nil {
		return nil, utils.LavaFormatError("getSessionFromAnActiveConsumer Failure", err, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "sessionId": strconv.FormatUint(sessionId, 10)})
	}

	if singleProviderSession.RelayNum+1 < relayNumber { // validate relay number here, but add only in PrepareSessionForUsage
		return nil, utils.LavaFormatError("singleProviderSession.RelayNum mismatch, session out of sync", SessionOutOfSyncError, &map[string]string{"singleProviderSession.RelayNum": strconv.FormatUint(singleProviderSession.RelayNum+1, 10), "request.relayNumber": strconv.FormatUint(relayNumber, 10)})
	}
	// singleProviderSession is locked at this point.
	return singleProviderSession, err
}

func (psm *ProviderSessionManager) GetSession(address string, epoch uint64, sessionId uint64, relayNumber uint64) (*SingleProviderSession, error) {
	valid, _ := psm.IsValidEpoch(epoch)
	if valid { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}

	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}

	return psm.getSingleSessionFromProviderSessionWithConsumer(providerSessionWithConsumer, sessionId, epoch, relayNumber)
}

func (psm *ProviderSessionManager) registerNewSession(address string, epoch uint64, sessionId uint64, vrfPk *utils.VrfPubKey, maxCuForConsumer uint64) (*ProviderSessionsWithConsumer, error) {
	psm.lock.Lock()
	defer psm.lock.Unlock()

	valid, _ := psm.IsValidEpoch(epoch)
	if valid { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}

	mapOfProviderSessionsWithConsumer, foundEpochInMap := psm.sessionsWithAllConsumers[epoch]
	if !foundEpochInMap {
		mapOfProviderSessionsWithConsumer = make(map[string]*ProviderSessionsWithConsumer)
		psm.sessionsWithAllConsumers[epoch] = mapOfProviderSessionsWithConsumer
	}

	providerSessionWithConsumer, foundAddressInMap := mapOfProviderSessionsWithConsumer[address]
	if !foundAddressInMap {
		providerSessionWithConsumer = &ProviderSessionsWithConsumer{
			consumer: address,
			epochData: &ProviderSessionsEpochData{
				VrfPk:           vrfPk,
				MaxComputeUnits: maxCuForConsumer,
			},
		}
		mapOfProviderSessionsWithConsumer[address] = providerSessionWithConsumer
	}
	return providerSessionWithConsumer, nil
}

// TODO add vrfPk and Max compute units.
func (psm *ProviderSessionManager) RegisterProviderSessionWithConsumer(address string, epoch uint64, sessionId uint64, relayNumber uint64, vrfPk *utils.VrfPubKey, maxCuForConsumer uint64) (*SingleProviderSession, error) {
	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			providerSessionWithConsumer, err = psm.registerNewSession(address, epoch, sessionId, vrfPk, maxCuForConsumer)
			if err != nil {
				return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed to registerNewSession", err, nil)
			}
		} else {
			return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed", err, nil)
		}
	}
	return psm.getSingleSessionFromProviderSessionWithConsumer(providerSessionWithConsumer, sessionId, epoch, relayNumber)
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
			if providerSessionWithConsumer.atomicReadConsumerBlocked() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "ConsumerAddress": address})
				return nil, ConsumerIsBlockListed
			}
			return providerSessionWithConsumer, nil // no error
		}
	}
	return nil, ConsumerNotRegisteredYet
}

func (psm *ProviderSessionManager) getSessionFromAnActiveConsumer(providerSessionWithConsumer *ProviderSessionsWithConsumer, sessionId uint64, epoch uint64) (singleProviderSession *SingleProviderSession, err error) {
	session, err := providerSessionWithConsumer.GetExistingSession(sessionId)
	if err == nil {
		return session, nil
	} else if SessionDoesNotExist.Is(err) {
		// if we don't have a session we need to create a new one.
		return providerSessionWithConsumer.createNewSingleProviderSession(sessionId, epoch)
	} else {
		utils.LavaFormatFatal("GetExistingSession Unexpected Error", err, nil)
		return nil, err
	}
}

func (psm *ProviderSessionManager) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil // TBD
}

func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64) (*SingleProviderSession, error) {
	return nil, fmt.Errorf("not implemented")
}

// OnSessionDone unlocks the session gracefully, this happens when session finished with an error
func (psm *ProviderSessionManager) OnSessionFailure(singleProviderSession *SingleProviderSession) (err error) {
	return singleProviderSession.onSessionFailure()
}

// OnSessionDone unlocks the session gracefully, this happens when session finished successfully
func (psm *ProviderSessionManager) OnSessionDone(singleProviderSession *SingleProviderSession) (err error) {
	return singleProviderSession.onSessionDone()
}

func (psm *ProviderSessionManager) RPCProviderEndpoint() *RPCProviderEndpoint {
	return psm.rpcProviderEndpoint
}

func (psm *ProviderSessionManager) UpdateEpoch(epoch uint64) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if epoch > psm.blockDistanceForEpochValidity {
		psm.blockedEpochHeight = epoch - psm.blockDistanceForEpochValidity
	} else {
		psm.blockedEpochHeight = 0
	}
	newMap := make(map[uint64]map[string]*ProviderSessionsWithConsumer)
	// In order to avoid running over the map twice, (1. mark 2. delete.) better technique is to copy and filter
	// which has better O(n) vs O(2n)
	for epochStored, value := range psm.sessionsWithAllConsumers {
		if !IsEpochValidForUse(epochStored, psm.blockedEpochHeight) {
			// epoch is not valid so we dont keep its key in the new map
			continue
		}
		// if epochStored is ok, copy the value into the new map
		newMap[epochStored] = value
	}
	psm.sessionsWithAllConsumers = newMap
}

func (psm *ProviderSessionManager) ProcessUnsubscribeEthereum(subscriptionID string, consumerAddress sdk.AccAddress) error {
	return fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) ProcessUnsubscribeTendermint(apiName string, subscriptionID string, consumerAddress sdk.AccAddress) error {
	return fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) NewSubscription(consumerAddress string, epoch uint64, subscription *RPCSubscription) error {
	// return an error if subscriptionID exists
	// original code:
	// userSessions.Lock.Lock()
	// if _, ok := userSessions.Subs[subscriptionID]; ok {
	// 	return utils.LavaFormatError("SubscriptiodID: "+subscriptionID+"exists", nil, nil)
	// }
	// userSessions.Subs[subscriptionID] = &subscription{
	// 	id:                   subscriptionID,
	// 	sub:                  clientSub,
	// 	subscribeRepliesChan: subscribeRepliesChan,
	// }
	// userSessions.Lock.Unlock()
	return fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) SubscriptionFailure(consumerAddress string, epoch uint64, subscriptionID string) {
	// original code
	// userSessions.Lock.Lock()
	// 		if sub, ok := userSessions.Subs[subscriptionID]; ok {
	// 			sub.disconnect()
	// 			delete(userSessions.Subs, subscriptionID)
	// 		}
	// 		userSessions.Lock.Unlock()
}

// Called when the reward server has information on a higher cu proof and usage and this providerSessionsManager needs to sync up on it
func (psm *ProviderSessionManager) UpdateSessionCU(consumerAddress string, epoch uint64, sessionID uint64, newCU uint64) error {
	// load the session and update the CU inside
	psm.lock.Lock()
	defer psm.lock.Unlock()
	valid, _ := psm.IsValidEpoch(epoch)
	if valid { // checking again because we are now locked and epoch cant change now.
		return utils.LavaFormatError("UpdateSessionCU", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
	}

	providerSessionsWithConsumerMap, ok := psm.sessionsWithAllConsumers[epoch]
	if !ok {
		return utils.LavaFormatError("UpdateSessionCU Failed", EpochIsNotRegisteredError, &map[string]string{"epoch": strconv.FormatUint(epoch, 10)})
	}
	providerSessionWithConsumer, foundConsumer := providerSessionsWithConsumerMap[consumerAddress]
	if !foundConsumer {
		return utils.LavaFormatError("UpdateSessionCU Failed", ConsumerIsNotRegisteredError, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "consumer": consumerAddress})
	}

	usedCu := providerSessionWithConsumer.atomicReadUsedComputeUnits() // check used cu now
	if usedCu < newCU {
		// if newCU proof is higher than current state, update.
		providerSessionWithConsumer.atomicWriteUsedComputeUnits(newCU)
	}
	return nil
}

// Returning a new provider session manager
func NewProviderSessionManager(rpcProviderEndpoint *RPCProviderEndpoint, numberOfBlocksKeptInMemory uint64) *ProviderSessionManager {
	return &ProviderSessionManager{rpcProviderEndpoint: rpcProviderEndpoint, blockDistanceForEpochValidity: numberOfBlocksKeptInMemory}
}

func IsEpochValidForUse(targetEpoch uint64, blockedEpochHeight uint64) bool {
	return targetEpoch > blockedEpochHeight
}
