package lavasession

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers                map[uint64]sessionData      // first key is epochs, second key is a consumer address
	dataReliabilitySessionsWithAllConsumers map[uint64]sessionData      // separate handling of data reliability so later on we can use it outside of pairing, first key is epochs, second key is a consumer address
	subscriptionSessionsWithAllConsumers    map[uint64]subscriptionData // first key is an epoch, second key is a consumer address, third key is subscriptionId
	lock                                    sync.RWMutex
	blockedEpochHeight                      uint64 // requests from this epoch are blocked
	rpcProviderEndpoint                     *RPCProviderEndpoint
	blockDistanceForEpochValidity           uint64 // sessionsWithAllConsumers with epochs older than ((latest epoch) - numberOfBlocksKeptInMemory) are deleted.
}

func (psm *ProviderSessionManager) GetProviderIndexWithConsumer(epoch uint64, consumerAddress string) (int64, error) {
	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, consumerAddress)
	if err != nil {
		// if consumer not active maybe it has a DR session. so check there as well
		psm.lock.RLock()
		defer psm.lock.RUnlock()
		drSessionData, found := psm.dataReliabilitySessionsWithAllConsumers[epoch]
		if found {
			drProviderSessionWithConsumer, foundDrSession := drSessionData.sessionMap[consumerAddress]
			if foundDrSession {
				return drProviderSessionWithConsumer.atomicReadPairedProviders(), nil
			}
		}
		// we didn't find the consumer in both maps
		return IndexNotFound, CouldNotFindIndexAsConsumerNotYetRegisteredError
	}
	return providerSessionWithConsumer.atomicReadPairedProviders(), nil
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

func (psm *ProviderSessionManager) getSingleSessionFromProviderSessionWithConsumer(ctx context.Context, providerSessionsWithConsumer *ProviderSessionsWithConsumer, sessionId uint64, epoch uint64, relayNumber uint64) (*SingleProviderSession, error) {
	if providerSessionsWithConsumer.atomicReadConsumerBlocked() != notBlockListedConsumer {
		return nil, utils.LavaFormatError("This consumer address is blocked.", nil, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "consumer", Value: providerSessionsWithConsumer.consumerAddr})
	}
	// get a single session and lock it, for error it's not locked
	singleProviderSession, err := psm.getSessionFromAnActiveConsumer(ctx, providerSessionsWithConsumer, sessionId, epoch) // after getting session verify relayNum etc..
	if err != nil {
		return nil, utils.LavaFormatError("getSessionFromAnActiveConsumer Failure", err, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "sessionId", Value: sessionId})
	}
	if singleProviderSession.RelayNum+1 > relayNumber { // validate relay number here, but add only in PrepareSessionForUsage
		// unlock the session since we are returning an error
		defer singleProviderSession.lock.Unlock()
		return nil, utils.LavaFormatError("singleProviderSession.RelayNum mismatch, session out of sync", SessionOutOfSyncError, utils.Attribute{Key: "singleProviderSession.RelayNum", Value: singleProviderSession.RelayNum + 1}, utils.Attribute{Key: "request.relayNumber", Value: relayNumber})
	}
	// singleProviderSession is locked at this point.
	return singleProviderSession, nil
}

func (psm *ProviderSessionManager) getOrCreateDataReliabilitySessionWithConsumer(address string, epoch uint64, sessionId uint64, pairedProviders int64) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	if mapOfDataReliabilitySessionsWithConsumer, consumerFoundInEpoch := psm.dataReliabilitySessionsWithAllConsumers[epoch]; consumerFoundInEpoch {
		if providerSessionWithConsumer, consumerAddressFound := mapOfDataReliabilitySessionsWithConsumer.sessionMap[address]; consumerAddressFound {
			if providerSessionWithConsumer.atomicReadConsumerBlocked() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "ConsumerAddress", Value: address})
				return nil, ConsumerIsBlockListed
			}
			if pairedProviders != providerSessionWithConsumer.atomicReadPairedProviders() {
				return nil, ProviderIndexMisMatchError
			}
			return providerSessionWithConsumer, nil // no error
		}
	} else {
		// If Epoch is missing from map, create a new instance
		psm.dataReliabilitySessionsWithAllConsumers[epoch] = sessionData{sessionMap: make(map[string]*ProviderSessionsWithConsumer)}
	}

	// If we got here, we need to create a new instance for this consumer address.
	providerSessionWithConsumer = NewProviderSessionsWithConsumer(address, nil, isDataReliabilityPSWC, pairedProviders)
	psm.dataReliabilitySessionsWithAllConsumers[epoch].sessionMap[address] = providerSessionWithConsumer
	return providerSessionWithConsumer, nil
}

// GetDataReliabilitySession fetches a data reliability session
func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64, sessionId uint64, relayNumber uint64, pairedProviders int64) (*SingleProviderSession, error) {
	// validate Epoch
	if !psm.IsValidEpoch(epoch) { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		return nil, InvalidEpochError
	}

	// validate sessionId
	if sessionId > DataReliabilitySessionId {
		return nil, utils.LavaFormatError("request's sessionId is larger than the data reliability allowed session ID", nil,
			utils.Attribute{Key: "sessionId", Value: sessionId},
			utils.Attribute{Key: "DataReliabilitySessionId", Value: strconv.Itoa(DataReliabilitySessionId)},
		)
	}

	// validate RelayNumber
	if relayNumber == 0 {
		return nil, utils.LavaFormatError("request's relayNumber zero, expecting consumer to increment", nil,
			utils.Attribute{Key: "relayNumber", Value: relayNumber},
			utils.Attribute{Key: "DataReliabilityRelayNumber", Value: DataReliabilityRelayNumber},
		)
	}

	if relayNumber > DataReliabilityRelayNumber {
		return nil, utils.LavaFormatError("request's relayNumber is larger than the DataReliabilityRelayNumber allowed in Data Reliability", nil,
			utils.Attribute{Key: "relayNumber", Value: relayNumber},
			utils.Attribute{Key: "DataReliabilityRelayNumber", Value: DataReliabilityRelayNumber},
		)
	}

	// validate active consumer.
	providerSessionWithConsumer, err := psm.getOrCreateDataReliabilitySessionWithConsumer(address, epoch, sessionId, pairedProviders)
	if err != nil {
		return nil, utils.LavaFormatError("getOrCreateDataReliabilitySessionWithConsumer Failed", err,
			utils.Attribute{Key: "relayNumber", Value: relayNumber},
			utils.Attribute{Key: "DataReliabilityRelayNumber", Value: DataReliabilityRelayNumber},
		)
	}

	// singleProviderSession is locked after this method is called unless we got an error
	singleProviderSession, err := providerSessionWithConsumer.getDataReliabilitySingleSession(sessionId, epoch)
	if err != nil {
		return nil, err
	}

	// validate relay number in the session stored
	if singleProviderSession.RelayNum+1 > DataReliabilityRelayNumber { // validate relay number fits if it has been used already raise a used error
		defer singleProviderSession.lock.Unlock() // in case of an error we need to unlock the session as its currently locked.
		return nil, utils.LavaFormatWarning("Data Reliability Session was already used", DataReliabilitySessionAlreadyUsedError)
	}

	return singleProviderSession, nil
}

func (psm *ProviderSessionManager) GetSession(ctx context.Context, address string, epoch uint64, sessionId uint64, relayNumber uint64, badge *pairingtypes.Badge) (*SingleProviderSession, error) {
	if !psm.IsValidEpoch(epoch) { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "blockedEpochHeight", Value: psm.blockedEpochHeight}, utils.Attribute{Key: "blockDistanceForEpochValidity", Value: psm.blockDistanceForEpochValidity})
		return nil, InvalidEpochError
	}

	providerSessionsWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}

	badgeUserEpochData := getOrCreateBadgeUserEpochData(badge, providerSessionsWithConsumer)
	singleProviderSession, err := psm.getSingleSessionFromProviderSessionWithConsumer(ctx, providerSessionsWithConsumer, sessionId, epoch, relayNumber)
	if badgeUserEpochData != nil {
		singleProviderSession.BadgeUserData = badgeUserEpochData
	}
	return singleProviderSession, err
}

func getOrCreateBadgeUserEpochData(badge *pairingtypes.Badge, providerSessionsWithConsumer *ProviderSessionsWithConsumer) (badgeUserEpochData *ProviderSessionsEpochData) {
	if badge == nil {
		return nil
	}
	badgeUserEpochData, exists := getBadgeEpochDataFromProviderSessionWithConsumer(badge.Address, providerSessionsWithConsumer)
	if !exists { // badgeUserEpochData not found, needs to be registered
		badgeUserEpochData = registerBadgeEpochDataToProviderSessionWithConsumer(badge.Address, badge.CuAllocation, providerSessionsWithConsumer)
	}
	return badgeUserEpochData
}

func getBadgeEpochDataFromProviderSessionWithConsumer(badgeUser string, providerSessionsWithConsumer *ProviderSessionsWithConsumer) (*ProviderSessionsEpochData, bool) {
	providerSessionsWithConsumer.Lock.RLock()
	defer providerSessionsWithConsumer.Lock.RUnlock()
	badgeUserEpochData, exists := providerSessionsWithConsumer.badgeEpochData[badgeUser]
	return badgeUserEpochData, exists
}

func registerBadgeEpochDataToProviderSessionWithConsumer(badgeUser string, badgeCuAllocation uint64, providerSessionsWithConsumer *ProviderSessionsWithConsumer) *ProviderSessionsEpochData {
	providerSessionsWithConsumer.Lock.Lock()
	defer providerSessionsWithConsumer.Lock.Unlock()
	providerSessionsWithConsumer.badgeEpochData[badgeUser] = &ProviderSessionsEpochData{MaxComputeUnits: slices.Min([]uint64{providerSessionsWithConsumer.epochData.MaxComputeUnits, badgeCuAllocation})}
	return providerSessionsWithConsumer.badgeEpochData[badgeUser]
}

func (psm *ProviderSessionManager) registerNewConsumer(consumerAddr string, epoch uint64, maxCuForConsumer uint64, pairedProviders int64) (*ProviderSessionsWithConsumer, error) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		return nil, InvalidEpochError
	}

	mapOfProviderSessionsWithConsumer, foundEpochInMap := psm.sessionsWithAllConsumers[epoch]
	if !foundEpochInMap {
		mapOfProviderSessionsWithConsumer = sessionData{sessionMap: make(map[string]*ProviderSessionsWithConsumer)}
		psm.sessionsWithAllConsumers[epoch] = mapOfProviderSessionsWithConsumer
	}

	providerSessionWithConsumer, foundAddressInMap := mapOfProviderSessionsWithConsumer.sessionMap[consumerAddr]
	if !foundAddressInMap {
		epochData := &ProviderSessionsEpochData{MaxComputeUnits: maxCuForConsumer}
		providerSessionWithConsumer = NewProviderSessionsWithConsumer(consumerAddr, epochData, notDataReliabilityPSWC, pairedProviders)
		mapOfProviderSessionsWithConsumer.sessionMap[consumerAddr] = providerSessionWithConsumer
	}

	return providerSessionWithConsumer, nil
}

func (psm *ProviderSessionManager) RegisterProviderSessionWithConsumer(ctx context.Context, consumerAddress string, epoch uint64, sessionId uint64, relayNumber uint64, maxCuForConsumer uint64, pairedProviders int64, badge *pairingtypes.Badge) (*SingleProviderSession, error) {
	_, err := psm.IsActiveConsumer(epoch, consumerAddress)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			_, err = psm.registerNewConsumer(consumerAddress, epoch, maxCuForConsumer, pairedProviders)
			if err != nil {
				return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed to registerNewSession", err)
			}
		} else {
			return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed", err)
		}
		utils.LavaFormatDebug("provider registered consumer", utils.Attribute{Key: "consumer", Value: consumerAddress}, utils.Attribute{Key: "epoch", Value: epoch})
	}
	return psm.GetSession(ctx, consumerAddress, epoch, sessionId, relayNumber, badge)
}

func (psm *ProviderSessionManager) getActiveConsumer(epoch uint64, address string) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		return nil, InvalidEpochError
	}
	if mapOfProviderSessionsWithConsumer, consumerFoundInEpoch := psm.sessionsWithAllConsumers[epoch]; consumerFoundInEpoch {
		if providerSessionWithConsumer, consumerAddressFound := mapOfProviderSessionsWithConsumer.sessionMap[address]; consumerAddressFound {
			if providerSessionWithConsumer.atomicReadConsumerBlocked() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "ConsumerAddress", Value: address})
				return nil, ConsumerIsBlockListed
			}
			return providerSessionWithConsumer, nil // no error
		}
	}
	return nil, ConsumerNotRegisteredYet
}

func (psm *ProviderSessionManager) getSessionFromAnActiveConsumer(ctx context.Context, providerSessionsWithConsumer *ProviderSessionsWithConsumer, sessionId uint64, epoch uint64) (singleProviderSession *SingleProviderSession, err error) {
	session, err := providerSessionsWithConsumer.getExistingSession(ctx, sessionId)
	if err == nil {
		return session, nil
	} else if SessionDoesNotExist.Is(err) {
		// if we don't have a session we need to create a new one.
		return providerSessionsWithConsumer.createNewSingleProviderSession(ctx, sessionId, epoch)
	} else {
		return nil, utils.LavaFormatError("could not get existing session", err, utils.Attribute{Key: "sessionId", Value: sessionId})
	}
}

func (psm *ProviderSessionManager) ReportConsumer() (address string, epoch uint64, err error) {
	return "", 0, nil // TBD
}

// OnSessionDone unlocks the session gracefully, this happens when session finished with an error
func (psm *ProviderSessionManager) OnSessionFailure(singleProviderSession *SingleProviderSession, relayNumber uint64) (err error) {
	if !psm.IsValidEpoch(singleProviderSession.PairingEpoch) {
		// the single provider session is no longer valid, so do not do a onSessionFailure, we don;t want it racing with cleanup touching other objects
		utils.LavaFormatWarning("epoch changed during session usage, so discarding sessionID changes on failure", nil,
			utils.Attribute{Key: "sessionID", Value: singleProviderSession.SessionID},
			utils.Attribute{Key: "cuSum", Value: singleProviderSession.CuSum},
			utils.Attribute{Key: "PairingEpoch", Value: singleProviderSession.PairingEpoch})
		return singleProviderSession.onSessionDone(relayNumber) // to unlock it and resume
	}
	return singleProviderSession.onSessionFailure()
}

// OnSessionDone unlocks the session gracefully, this happens when session finished successfully
func (psm *ProviderSessionManager) OnSessionDone(singleProviderSession *SingleProviderSession, relayNumber uint64) (err error) {
	return singleProviderSession.onSessionDone(relayNumber)
}

func (psm *ProviderSessionManager) RPCProviderEndpoint() *RPCProviderEndpoint {
	return psm.rpcProviderEndpoint
}

// on a new epoch we are cleaning stale provider data, also we are making sure consumers who are trying to use past data are not capable to
func (psm *ProviderSessionManager) UpdateEpoch(epoch uint64) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if epoch <= psm.blockedEpochHeight {
		// this shouldn't happen, but nothing to do
		utils.LavaFormatWarning("called updateEpoch with invalid epoch", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "blockedEpoch", Value: psm.blockedEpochHeight})
		return
	}
	if epoch > psm.blockDistanceForEpochValidity {
		psm.blockedEpochHeight = epoch - psm.blockDistanceForEpochValidity
	} else {
		psm.blockedEpochHeight = 0
	}
	psm.sessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.sessionsWithAllConsumers)
	psm.dataReliabilitySessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.dataReliabilitySessionsWithAllConsumers)
	psm.subscriptionSessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.subscriptionSessionsWithAllConsumers)
}

func filterOldEpochEntries[T dataHandler](blockedEpochHeight uint64, allEpochsMap map[uint64]T) (validEpochsMap map[uint64]T) {
	// In order to avoid running over the map twice, (1. mark 2. delete.) better technique is to copy and filter
	// which has better O(n) vs O(2n)
	validEpochsMap = map[uint64]T{}
	for epochStored, value := range allEpochsMap {
		if !IsEpochValidForUse(epochStored, blockedEpochHeight) {
			// epoch is not valid so we don't keep its key in the new map

			// in the case of subscribe, we need to unsubscribe before deleting the key from storage.
			value.onDeleteEvent()

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
		return utils.LavaFormatError("Couldn't find epoch in psm.subscriptionSessionsWithAllConsumers", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "address", Value: consumerAddress})
	}
	mapOfSubscriptionId, foundMapOfSubscriptionId := mapOfConsumers.subscriptionMap[consumerAddress]
	if !foundMapOfSubscriptionId {
		return utils.LavaFormatError("Couldn't find consumer address in psm.subscriptionSessionsWithAllConsumers", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "address", Value: consumerAddress})
	}

	var err error
	if apiName == TendermintUnsubscribeAll {
		// unsubscribe all subscriptions
		for _, v := range mapOfSubscriptionId {
			if v.Sub == nil {
				err = utils.LavaFormatError("ProcessUnsubscribe TendermintUnsubscribeAll mapOfSubscriptionId Error", SubscriptionPointerIsNilError, utils.Attribute{Key: "subscripionId", Value: subscriptionID})
			} else {
				v.Sub.Unsubscribe()
			}
		}
		psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress] = make(map[string]*RPCSubscription) // delete the entire map.
		return err
	}

	subscription, foundSubscription := mapOfSubscriptionId[subscriptionID]
	if !foundSubscription {
		return utils.LavaFormatError("Couldn't find subscription Id in psm.subscriptionSessionsWithAllConsumers", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "address", Value: consumerAddress}, utils.Attribute{Key: "subscriptionId", Value: subscriptionID})
	}

	if subscription.Sub == nil {
		err = utils.LavaFormatError("ProcessUnsubscribe Error", SubscriptionPointerIsNilError, utils.Attribute{Key: "subscripionId", Value: subscriptionID})
	} else {
		subscription.Sub.Unsubscribe()
	}
	delete(psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress], subscriptionID) // delete subscription after finished with it
	return err
}

func (psm *ProviderSessionManager) addSubscriptionToStorage(subscription *RPCSubscription, consumerAddress string, epoch uint64) error {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	// we already validated the epoch is valid in the GetSessions no need to verify again.
	_, foundEpoch := psm.subscriptionSessionsWithAllConsumers[epoch]
	if !foundEpoch {
		// this is the first time we subscribe in this epoch
		psm.subscriptionSessionsWithAllConsumers[epoch] = subscriptionData{subscriptionMap: make(map[string]map[string]*RPCSubscription)}
	}

	_, foundSubscriptions := psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress]
	if !foundSubscriptions {
		// this is the first subscription added in this epoch. we need to create the map
		psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress] = make(map[string]*RPCSubscription)
	}

	_, foundSubscription := psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress][subscription.Id]
	if !foundSubscription {
		// we shouldnt find a subscription already in the storage.
		psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress][subscription.Id] = subscription
		return nil // successfully added subscription to storage
	}

	// if we get here we found a subscription already in the storage and we need to return an error as we can't add two subscriptions with the same id
	return utils.LavaFormatError("addSubscription", SubscriptionAlreadyExistsError, utils.Attribute{Key: "SubscriptionId", Value: subscription.Id}, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "address", Value: consumerAddress})
}

func (psm *ProviderSessionManager) ReleaseSessionAndCreateSubscription(session *SingleProviderSession, subscription *RPCSubscription, consumerAddress string, epoch uint64, relayNumber uint64) error {
	err := psm.OnSessionDone(session, relayNumber)
	if err != nil {
		return utils.LavaFormatError("Failed ReleaseSessionAndCreateSubscription", err)
	}
	return psm.addSubscriptionToStorage(subscription, consumerAddress, epoch)
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
	mapOfSubscriptionId, foundMapOfSubscriptionId := mapOfConsumers.subscriptionMap[consumerAddress]
	if !foundMapOfSubscriptionId {
		return
	}

	subscription, foundSubscription := mapOfSubscriptionId[subscriptionID]
	if !foundSubscription {
		return
	}

	if subscription.Sub == nil { // validate subscription not nil
		utils.LavaFormatError("SubscriptionEnded Error", SubscriptionPointerIsNilError, utils.Attribute{Key: "subscripionId", Value: subscription.Id})
	} else {
		subscription.Sub.Unsubscribe()
	}
	delete(psm.subscriptionSessionsWithAllConsumers[epoch].subscriptionMap[consumerAddress], subscriptionID) // delete subscription after finished with it
}

// Called when the reward server has information on a higher cu proof and usage and this providerSessionsManager needs to sync up on it
func (psm *ProviderSessionManager) UpdateSessionCU(consumerAddress string, epoch uint64, sessionID uint64, newCU uint64) error {
	// load the session and update the CU inside

	// Step 1: Lock psm
	getProviderSessionsFromManager := func() (*ProviderSessionsWithConsumer, error) {
		psm.lock.RLock()
		defer psm.lock.RUnlock()
		if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
			return nil, utils.LavaFormatError("UpdateSessionCU", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		}

		providerSessionsWithConsumerMap, ok := psm.sessionsWithAllConsumers[epoch]
		if !ok {
			return nil, utils.LavaFormatError("UpdateSessionCU Failed", EpochIsNotRegisteredError, utils.Attribute{Key: "epoch", Value: epoch})
		}

		providerSessionsWithConsumer, foundConsumer := providerSessionsWithConsumerMap.sessionMap[consumerAddress]
		if !foundConsumer {
			return nil, utils.LavaFormatError("UpdateSessionCU Failed", ConsumerIsNotRegisteredError, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "consumer", Value: consumerAddress})
		}

		return providerSessionsWithConsumer, nil
	}
	providerSessionsWithConsumer, err := getProviderSessionsFromManager()
	if err != nil {
		return err
	}
	// Step 2: Lock pswc and get the session id
	getSessionIDFromProviderSessions := func() (*SingleProviderSession, error) {
		providerSessionsWithConsumer.Lock.RLock()
		defer providerSessionsWithConsumer.Lock.RUnlock()
		singleSession, foundSession := providerSessionsWithConsumer.Sessions[sessionID]
		if !foundSession {
			return nil, utils.LavaFormatError("UpdateSessionCU Failed", SessionIdNotFoundError, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "consumer", Value: consumerAddress}, utils.Attribute{Key: "sessionId", Value: sessionID})
		}

		return singleSession, nil
	}
	singleSession, err := getSessionIDFromProviderSessions()
	if err != nil {
		return err
	}
	// Step 3: update information atomically. ( no locks required when updating atomically )
	oldCuSum := singleSession.atomicReadCuSum() // check used cu now
	if newCU > oldCuSum {
		// TODO: use compare and swap for race avoidance
		// update the session.
		singleSession.writeCuSumAtomically(newCU)
		usedCuInParent := providerSessionsWithConsumer.atomicReadUsedComputeUnits()
		usedCuInParent += (newCU - oldCuSum) // new cu is larger than old cu. so its ok to subtract
		// update the parent
		providerSessionsWithConsumer.atomicWriteUsedComputeUnits(usedCuInParent)
	}
	return nil
}

// Returning a new provider session manager
func NewProviderSessionManager(rpcProviderEndpoint *RPCProviderEndpoint, numberOfBlocksKeptInMemory uint64) *ProviderSessionManager {
	return &ProviderSessionManager{
		rpcProviderEndpoint:                     rpcProviderEndpoint,
		blockDistanceForEpochValidity:           numberOfBlocksKeptInMemory,
		sessionsWithAllConsumers:                map[uint64]sessionData{},
		dataReliabilitySessionsWithAllConsumers: map[uint64]sessionData{},
		subscriptionSessionsWithAllConsumers:    map[uint64]subscriptionData{},
	}
}

func IsEpochValidForUse(targetEpoch uint64, blockedEpochHeight uint64) bool {
	return targetEpoch > blockedEpochHeight
}
