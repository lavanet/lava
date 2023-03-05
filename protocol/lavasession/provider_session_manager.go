package lavasession

import (
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/utils"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers                map[uint64]map[string]*ProviderSessionsWithConsumer // first key is epochs, second key is a consumer address
	dataReliabilitySessionsWithAllConsumers map[uint64]map[string]*ProviderSessionsWithConsumer // separate handling of data reliability so later on we can use it outside of pairing, first key is epochs, second key is a consumer address
	subscriptionSessionsWithAllConsumers    map[uint64]map[string]map[string]*RPCSubscription   // first key is an epoch, second key is a consumer address, third key is subscriptionId
	lock                                    sync.RWMutex
	blockedEpochHeight                      uint64 // requests from this epoch are blocked
	rpcProviderEndpoint                     *RPCProviderEndpoint
	blockDistanceForEpochValidity           uint64 // sessionsWithAllConsumers with epochs older than ((latest epoch) - numberOfBlocksKeptInMemory) are deleted.
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicWriteBlockedEpoch(epoch uint64) {
	atomic.StoreUint64(&psm.blockedEpochHeight, epoch)
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicReadBlockedEpoch() (epoch uint64) {
	return atomic.LoadUint64(&psm.blockedEpochHeight)
}

func (psm *ProviderSessionManager) GetBlockedEpochHeight() uint64 {
	return psm.atomicReadBlockedEpoch()
}

func (psm *ProviderSessionManager) IsValidEpoch(epoch uint64) (valid bool) {
	return epoch > psm.atomicReadBlockedEpoch()
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

func (psm *ProviderSessionManager) getOrCreateDataReliabilitySessionWithConsumer(address string, epoch uint64, sessionId uint64) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	if mapOfDataReliabilitySessionsWithConsumer, consumerFoundInEpoch := psm.dataReliabilitySessionsWithAllConsumers[epoch]; consumerFoundInEpoch {
		if providerSessionWithConsumer, consumerAddressFound := mapOfDataReliabilitySessionsWithConsumer[address]; consumerAddressFound {
			if providerSessionWithConsumer.atomicReadConsumerBlocked() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "ConsumerAddress": address})
				return nil, ConsumerIsBlockListed
			}
			return providerSessionWithConsumer, nil // no error
		}
	} else {
		// If Epoch is missing from map, create a new instance
		psm.dataReliabilitySessionsWithAllConsumers[epoch] = make(map[string]*ProviderSessionsWithConsumer)
	}

	// If we got here, we need to create a new instance for this consumer address.
	providerSessionWithConsumer = &ProviderSessionsWithConsumer{
		consumer: address,
	}
	psm.dataReliabilitySessionsWithAllConsumers[epoch][address] = providerSessionWithConsumer
	return providerSessionWithConsumer, nil
}

// GetDataReliabilitySession fetches a data reliability session, and assumes the user
func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64, sessionId uint64, relayNumber uint64) (*SingleProviderSession, error) {
	// validate Epoch
	if !psm.IsValidEpoch(epoch) { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}

	// validate sessionId
	if sessionId > DataReliabilitySessionId {
		return nil, utils.LavaFormatError("request's sessionId is larger than the data reliability allowed session ID", nil, &map[string]string{"sessionId": strconv.FormatUint(sessionId, 10), "DataReliabilitySessionId": strconv.Itoa(DataReliabilitySessionId)})
	}

	// validate RelayNumber
	if relayNumber > DataReliabilityRelayNumber {
		return nil, utils.LavaFormatError("request's relayNumber is larger than the DataReliabilityRelayNumber allowed relay number", nil, &map[string]string{"relayNumber": strconv.FormatUint(relayNumber, 10), "DataReliabilityRelayNumber": strconv.Itoa(DataReliabilityRelayNumber)})
	}

	// validate active consumer.
	psm.getOrCreateDataReliabilitySessionWithConsumer(address, epoch, sessionId)

	return nil, nil

}

func (psm *ProviderSessionManager) GetSession(address string, epoch uint64, sessionId uint64, relayNumber uint64) (*SingleProviderSession, error) {
	if !psm.IsValidEpoch(epoch) { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10), "blockedEpochHeight": strconv.FormatUint(psm.blockedEpochHeight, 10), "blockDistanceForEpochValidity": strconv.FormatUint(psm.blockDistanceForEpochValidity, 10)})
		return nil, InvalidEpochError
	}

	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}

	return psm.getSingleSessionFromProviderSessionWithConsumer(providerSessionWithConsumer, sessionId, epoch, relayNumber)
}

func (psm *ProviderSessionManager) registerNewSession(address string, epoch uint64, sessionId uint64, maxCuForConsumer uint64) (*ProviderSessionsWithConsumer, error) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
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
				MaxComputeUnits: maxCuForConsumer,
			},
		}
		mapOfProviderSessionsWithConsumer[address] = providerSessionWithConsumer
	}
	return providerSessionWithConsumer, nil
}

func (psm *ProviderSessionManager) RegisterProviderSessionWithConsumer(address string, epoch uint64, sessionId uint64, relayNumber uint64, maxCuForConsumer uint64) (*SingleProviderSession, error) {
	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			providerSessionWithConsumer, err = psm.registerNewSession(address, epoch, sessionId, maxCuForConsumer)
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
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, &map[string]string{"RequestedEpoch": strconv.FormatUint(epoch, 10)})
		return nil, InvalidEpochError
	}
	if mapOfProviderSessionsWithConsumer, consumerFoundInEpoch := psm.sessionsWithAllConsumers[epoch]; consumerFoundInEpoch {
		if providerSessionWithConsumer, consumerAddressFound := mapOfProviderSessionsWithConsumer[address]; consumerAddressFound {
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

// on a new epoch we are cleaning stale provider data, also we are making sure consumers who are trying to use past data are not capable to
func (psm *ProviderSessionManager) UpdateEpoch(epoch uint64) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if epoch > psm.blockDistanceForEpochValidity {
		psm.blockedEpochHeight = epoch - psm.blockDistanceForEpochValidity
	} else {
		psm.blockedEpochHeight = 0
	}
	psm.sessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.sessionsWithAllConsumers)
	psm.dataReliabilitySessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.dataReliabilitySessionsWithAllConsumers)
	psm.subscriptionSessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.subscriptionSessionsWithAllConsumers)
}

func filterOldEpochEntries[T any](blockedEpochHeight uint64, allEpochsMap map[uint64]T) (validEpochsMap map[uint64]T) {
	// In order to avoid running over the map twice, (1. mark 2. delete.) better technique is to copy and filter
	// which has better O(n) vs O(2n)
	validEpochsMap = map[uint64]T{}
	for epochStored, value := range allEpochsMap {
		if !IsEpochValidForUse(epochStored, blockedEpochHeight) {
			// epoch is not valid so we don't keep its key in the new map
			continue
		}
		// if epochStored is ok, copy the value into the new map
		validEpochsMap[epochStored] = value
	}
	return
}

func (psm *ProviderSessionManager) ProcessUnsubscribe(apiName string, subscriptionID string, consumerAddress string, epoch uint64) error {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	mapOfConsumers, foundMapOfConsumers := psm.subscriptionSessionsWithAllConsumers[epoch]
	if !foundMapOfConsumers {
		return utils.LavaFormatError("Couldn't find epoch in psm.subscriptionSessionsWithAllConsumers", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "address": consumerAddress})
	}
	mapOfSubscriptionId, foundMapOfSubscriptionId := mapOfConsumers[consumerAddress]
	if !foundMapOfSubscriptionId {
		return utils.LavaFormatError("Couldn't find consumer address in psm.subscriptionSessionsWithAllConsumers", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "address": consumerAddress})
	}

	if apiName == TendermintUnsubscribeAll {
		// unsubscribe all subscriptions
		for _, v := range mapOfSubscriptionId {
			v.Sub.Unsubscribe()
		}
		return nil
	}

	subscription, foundSubscription := mapOfSubscriptionId[subscriptionID]
	if !foundSubscription {
		return utils.LavaFormatError("Couldn't find subscription Id in psm.subscriptionSessionsWithAllConsumers", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "address": consumerAddress, "subscriptionId": subscriptionID})
	}
	subscription.Sub.Unsubscribe()
	delete(mapOfSubscriptionId, subscriptionID) // delete subscription after finished with it
	return nil
}

func (psm *ProviderSessionManager) ReleaseSessionAndCreateSubscription(session *SingleProviderSession, subscription *RPCSubscription, consumerAddress string, epoch uint64) error {
	err := psm.OnSessionDone(session)
	if err != nil {
		return utils.LavaFormatError("Failed ReleaseSessionAndCreateSubscription", err, nil)
	}
	return nil
}

// try to disconnect the subscription incase we got an error.
// if fails to find assumes it was unsubscribed normally
func (psm *ProviderSessionManager) SubscriptionEnded(consumerAddress string, epoch uint64, subscriptionID string) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	mapOfConsumers, foundMapOfConsumers := psm.subscriptionSessionsWithAllConsumers[epoch]
	if !foundMapOfConsumers {
		return
	}
	mapOfSubscriptionId, foundMapOfSubscriptionId := mapOfConsumers[consumerAddress]
	if !foundMapOfSubscriptionId {
		return
	}

	subscription, foundSubscription := mapOfSubscriptionId[subscriptionID]
	if !foundSubscription {
		return
	}
	subscription.Sub.Unsubscribe()
	delete(mapOfSubscriptionId, subscriptionID) // delete subscription after finished with it
}

// Called when the reward server has information on a higher cu proof and usage and this providerSessionsManager needs to sync up on it
func (psm *ProviderSessionManager) UpdateSessionCU(consumerAddress string, epoch uint64, sessionID uint64, newCU uint64) error {
	// load the session and update the CU inside
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
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
	return &ProviderSessionManager{
		rpcProviderEndpoint:                     rpcProviderEndpoint,
		blockDistanceForEpochValidity:           numberOfBlocksKeptInMemory,
		sessionsWithAllConsumers:                map[uint64]map[string]*ProviderSessionsWithConsumer{},
		dataReliabilitySessionsWithAllConsumers: map[uint64]map[string]*ProviderSessionsWithConsumer{},
		subscriptionSessionsWithAllConsumers:    map[uint64]map[string]map[string]*RPCSubscription{},
	}
}

func IsEpochValidForUse(targetEpoch uint64, blockedEpochHeight uint64) bool {
	return targetEpoch > blockedEpochHeight
}
