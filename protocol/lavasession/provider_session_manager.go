package lavasession

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers map[uint64]map[string]*ProviderSessionsWithConsumer // first key is epochs, second key is a consumer address
	lock                     sync.RWMutex
	blockedEpoch             uint64 // requests from this epoch are blocked
	rpcProviderEndpoint      *RPCProviderEndpoint
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
func (psm *ProviderSessionManager) IsActiveConsumer(epoch uint64, address string) (providerSessionWithConsumer *ProviderSessionsWithConsumer, err error) {
	providerSessionWithConsumer, err = psm.getActiveConsumer(epoch, address)
	if err != nil {
		return nil, err
	}
	return providerSessionWithConsumer, nil // no error
}

func (psm *ProviderSessionManager) getSingleSessionFromProviderSessionWithConsumer(providerSessionWithConsumer *ProviderSessionsWithConsumer, sessionId uint64, epoch uint64, relayNumber uint64) (*SingleProviderSession, error) {
	// TODO:: we can validate here if consumer is blocked with atomicWriteBlockedEpoch
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

func (psm *ProviderSessionManager) registerNewSession(address string, epoch uint64, sessionId uint64) (*ProviderSessionsWithConsumer, error) {
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
			consumer:  address,
			epochData: &ProviderSessionsEpochData{}, // TODO add here all the epoch data get from user
		}
		mapOfProviderSessionsWithConsumer[address] = providerSessionWithConsumer
	}
	return providerSessionWithConsumer, nil
}

// TODO add vrfPk and Max compute units.
func (psm *ProviderSessionManager) RegisterProviderSessionWithConsumer(address string, epoch uint64, sessionId uint64, relayNumber uint64) (*SingleProviderSession, error) {
	providerSessionWithConsumer, err := psm.IsActiveConsumer(epoch, address)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			providerSessionWithConsumer, err = psm.registerNewSession(address, epoch, sessionId)
			if err != nil {
				return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed to registerNewSession", err, nil)
			}
		} else {
			return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed", err, nil)
		}
	}
	// validate relay number?? == 1
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
	return "", 0, nil
}

func (psm *ProviderSessionManager) GetDataReliabilitySession(address string, epoch uint64) (*SingleProviderSession, error) {
	return nil, fmt.Errorf("not implemented")
}

func (psm *ProviderSessionManager) OnSessionFailure(singleProviderSession *SingleProviderSession) (err error) {
	// need to handle dataReliability session failure separately
	return nil
}

func (psm *ProviderSessionManager) OnSessionDone(singleProviderSession *SingleProviderSession, request *pairingtypes.RelayRequest) (err error) {
	// need to handle dataReliability session separately
	// store the request as proof
	return nil
}

func (psm *ProviderSessionManager) RPCProviderEndpoint() *RPCProviderEndpoint {
	return psm.rpcProviderEndpoint
}

func (psm *ProviderSessionManager) UpdateEpoch(epoch uint64) {
	// update the epoch to limit consumer usage
	// when updating the blocked epoch, we also need to clean old epochs from the map. sessionsWithAllConsumers
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

// called when the reward server has information on a higher cu proof and usage and this providerSessionsManager needs to sync up on it
func (psm *ProviderSessionManager) UpdateSessionCU(consumerAddress string, epoch uint64, sessionID uint64, storedCU uint64) error {
	// load the session and update the CU inside
	return fmt.Errorf("not implemented")
}

// Returning a new provider session manager
func NewProviderSessionManager(rpcProviderEndpoint *RPCProviderEndpoint) *ProviderSessionManager {
	return &ProviderSessionManager{rpcProviderEndpoint: rpcProviderEndpoint}
}
