package lavasession

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ProviderSessionManager struct {
	sessionsWithAllConsumers      map[uint64]sessionData // first key is epochs, second key is a consumer address
	lock                          sync.RWMutex
	blockedEpochHeight            uint64 // requests from this epoch are blocked
	currentEpoch                  uint64
	rpcProviderEndpoint           *RPCProviderEndpoint
	blockDistanceForEpochValidity uint64                             // sessionsWithAllConsumers with epochs older than ((latest epoch) - numberOfBlocksKeptInMemory) are deleted.
	consumerPairedWithProjectMap  map[uint64]*projectConsumerMapping // consumer address as key, project as value
}

// reads cs.BlockedEpoch atomically
func (psm *ProviderSessionManager) atomicReadBlockedEpoch() (epoch uint64) {
	return atomic.LoadUint64(&psm.blockedEpochHeight)
}

func (psm *ProviderSessionManager) GetBlockedEpochHeight() uint64 {
	return psm.atomicReadBlockedEpoch()
}

func (psm *ProviderSessionManager) GetCurrentEpochAtomic() uint64 {
	return atomic.LoadUint64(&psm.currentEpoch)
}

func (psm *ProviderSessionManager) IsValidEpoch(epoch uint64) (valid bool) {
	return epoch > psm.atomicReadBlockedEpoch()
}

// Check if consumer exists and is not blocked, if all is valid return the ProviderSessionsWithConsumer pointer
func (psm *ProviderSessionManager) IsActiveProject(epoch uint64, projectId string) (providerSessionWithConsumer *ProviderSessionsWithConsumerProject, err error) {
	providerSessionWithConsumer, err = psm.getActiveProject(epoch, projectId)
	if err != nil {
		return nil, err
	}
	return providerSessionWithConsumer, nil // no error
}

func (psm *ProviderSessionManager) getSingleSessionFromProviderSessionWithConsumer(ctx context.Context, providerSessionsWithConsumer *ProviderSessionsWithConsumerProject, sessionId, epoch, relayNumber uint64) (*SingleProviderSession, error) {
	if providerSessionsWithConsumer.atomicReadConsumerBlocked() != notBlockListedConsumer {
		return nil, utils.LavaFormatError("This consumer address is blocked.", nil, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "consumer", Value: providerSessionsWithConsumer.consumersProjectId})
	}
	// get a single session and lock it, for error it's not locked
	singleProviderSession, err := psm.getSessionFromAnActiveConsumer(ctx, providerSessionsWithConsumer, sessionId, epoch) // after getting session verify relayNum etc..
	if err != nil {
		return nil, utils.LavaFormatError("getSessionFromAnActiveConsumer Failure", err, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "sessionId", Value: sessionId})
	}
	if singleProviderSession.RelayNum+1 > relayNumber { // validate relay number here, but add only in PrepareSessionForUsage
		// unlock the session since we are returning an error
		defer singleProviderSession.lock.Unlock()
		return nil, utils.LavaFormatError("singleProviderSession.RelayNum mismatch, session out of sync", SessionOutOfSyncError, utils.LogAttr("GUID", ctx), utils.LogAttr("errCount", singleProviderSession.errorsCount), utils.LogAttr("sessionID", singleProviderSession.SessionID), utils.Attribute{Key: "singleProviderSession.RelayNum", Value: singleProviderSession.RelayNum + 1}, utils.Attribute{Key: "request.relayNumber", Value: relayNumber})
	}
	// singleProviderSession is locked at this point.
	return singleProviderSession, nil
}

func (psm *ProviderSessionManager) GetSession(ctx context.Context, consumerAddress string, epoch, sessionId, relayNumber uint64, badge *pairingtypes.Badge) (*SingleProviderSession, error) {
	if !psm.IsValidEpoch(epoch) { // fast checking to see if epoch is even relevant
		utils.LavaFormatError("GetSession", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "blockedEpochHeight", Value: psm.blockedEpochHeight}, utils.Attribute{Key: "blockDistanceForEpochValidity", Value: psm.blockDistanceForEpochValidity})
		return nil, InvalidEpochError
	}

	projectId, found := psm.readConsumerToPairedWithProjectMap(consumerAddress, epoch)
	if !found {
		return nil, ConsumerNotRegisteredYet // if this consumer address was not matched with a projectId we need to register it
	}

	providerSessionsWithConsumer, err := psm.IsActiveProject(epoch, projectId)
	if err != nil {
		return nil, err
	}

	badgeUserEpochData := getOrCreateBadgeUserEpochData(badge, providerSessionsWithConsumer)
	singleProviderSession, err := psm.getSingleSessionFromProviderSessionWithConsumer(ctx, providerSessionsWithConsumer, sessionId, epoch, relayNumber)
	if badgeUserEpochData != nil && err == nil {
		singleProviderSession.BadgeUserData = badgeUserEpochData
	}
	return singleProviderSession, err
}

func getOrCreateBadgeUserEpochData(badge *pairingtypes.Badge, providerSessionsWithConsumer *ProviderSessionsWithConsumerProject) (badgeUserEpochData *ProviderSessionsEpochData) {
	if badge == nil {
		return nil
	}
	badgeUserEpochData, exists := getBadgeEpochDataFromProviderSessionWithConsumer(badge.Address, providerSessionsWithConsumer)
	if !exists { // badgeUserEpochData not found, needs to be registered
		badgeUserEpochData = registerBadgeEpochDataToProviderSessionWithConsumer(badge.Address, badge.CuAllocation, providerSessionsWithConsumer)
	}
	return badgeUserEpochData
}

func getBadgeEpochDataFromProviderSessionWithConsumer(badgeUser string, providerSessionsWithConsumer *ProviderSessionsWithConsumerProject) (*ProviderSessionsEpochData, bool) {
	providerSessionsWithConsumer.Lock.RLock()
	defer providerSessionsWithConsumer.Lock.RUnlock()
	badgeUserEpochData, exists := providerSessionsWithConsumer.badgeEpochData[badgeUser]
	return badgeUserEpochData, exists
}

func registerBadgeEpochDataToProviderSessionWithConsumer(badgeUser string, badgeCuAllocation uint64, providerSessionsWithConsumer *ProviderSessionsWithConsumerProject) *ProviderSessionsEpochData {
	providerSessionsWithConsumer.Lock.Lock()
	defer providerSessionsWithConsumer.Lock.Unlock()
	providerSessionsWithConsumer.badgeEpochData[badgeUser] = &ProviderSessionsEpochData{MaxComputeUnits: lavaslices.Min([]uint64{providerSessionsWithConsumer.epochData.MaxComputeUnits, badgeCuAllocation})}
	return providerSessionsWithConsumer.badgeEpochData[badgeUser]
}

func (psm *ProviderSessionManager) registerNewConsumer(consumerAddr string, projectId string, epoch, maxCuForConsumer uint64, pairedProviders int64) (*ProviderSessionsWithConsumerProject, error) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		return nil, InvalidEpochError
	}

	mapOfProviderSessionsWithConsumer, foundEpochInMap := psm.sessionsWithAllConsumers[epoch]
	if !foundEpochInMap {
		mapOfProviderSessionsWithConsumer = sessionData{sessionMap: make(map[string]*ProviderSessionsWithConsumerProject)}
		psm.sessionsWithAllConsumers[epoch] = mapOfProviderSessionsWithConsumer
	}

	providerSessionWithConsumer, foundAddressInMap := mapOfProviderSessionsWithConsumer.sessionMap[projectId]
	if !foundAddressInMap {
		epochData := &ProviderSessionsEpochData{MaxComputeUnits: maxCuForConsumer}
		providerSessionWithConsumer = NewProviderSessionsWithConsumer(projectId, epochData, notDataReliabilityPSWC, pairedProviders)
		mapOfProviderSessionsWithConsumer.sessionMap[projectId] = providerSessionWithConsumer
	}

	_, foundEpochInProjectMap := psm.consumerPairedWithProjectMap[epoch]
	if !foundEpochInProjectMap {
		psm.consumerPairedWithProjectMap[epoch] = &projectConsumerMapping{consumerToProjectMap: map[string]string{consumerAddr: projectId}}
	} else {
		psm.consumerPairedWithProjectMap[epoch].consumerToProjectMap[consumerAddr] = projectId
	}

	return providerSessionWithConsumer, nil
}

func (psm *ProviderSessionManager) readConsumerToPairedWithProjectMap(consumerAddress string, epoch uint64) (projectId string, found bool) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()

	_, foundEpoch := psm.consumerPairedWithProjectMap[epoch]
	if !foundEpoch {
		return "", false
	}

	project, foundConsumer := psm.consumerPairedWithProjectMap[epoch].consumerToProjectMap[consumerAddress]
	return project, foundConsumer
}

func (psm *ProviderSessionManager) writeConsumerToPairedWithProjectMap(consumerAddress, projectId string, epoch uint64) {
	psm.lock.Lock()
	defer psm.lock.Unlock()
	_, foundEpoch := psm.consumerPairedWithProjectMap[epoch]
	if !foundEpoch {
		utils.LavaFormatError("asked to write consumer to project map but epoch was missing, probably a bad flow", nil)
		psm.consumerPairedWithProjectMap[epoch] = &projectConsumerMapping{consumerToProjectMap: map[string]string{consumerAddress: projectId}}
	}
	psm.consumerPairedWithProjectMap[epoch].consumerToProjectMap[consumerAddress] = projectId
}

func (psm *ProviderSessionManager) RegisterProviderSessionWithConsumer(ctx context.Context, consumerAddress string, epoch, sessionId, relayNumber, maxCuForConsumer uint64, pairedProviders int64, projectId string, badge *pairingtypes.Badge) (*SingleProviderSession, error) {
	_, err := psm.IsActiveProject(epoch, projectId)
	if err != nil {
		if ConsumerNotRegisteredYet.Is(err) {
			_, err = psm.registerNewConsumer(consumerAddress, projectId, epoch, maxCuForConsumer, pairedProviders)
			if err != nil {
				return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed to registerNewSession", err)
			}
		} else {
			return nil, utils.LavaFormatError("RegisterProviderSessionWithConsumer Failed", err)
		}
		utils.LavaFormatDebug("provider registered consumer", utils.Attribute{Key: "consumer", Value: consumerAddress}, utils.Attribute{Key: "epoch", Value: epoch})
	} else {
		// skip lookup by just writing the consumer address directly
		// this flow happens only if the project ID was registered with another consumer.
		psm.writeConsumerToPairedWithProjectMap(consumerAddress, projectId, epoch)
	}
	return psm.GetSession(ctx, consumerAddress, epoch, sessionId, relayNumber, badge)
}

func (psm *ProviderSessionManager) getActiveProject(epoch uint64, projectId string) (providerSessionWithConsumer *ProviderSessionsWithConsumerProject, err error) {
	psm.lock.RLock()
	defer psm.lock.RUnlock()
	if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
		utils.LavaFormatError("getActiveConsumer", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		return nil, InvalidEpochError
	}
	if mapOfProviderSessionsWithConsumer, consumerFoundInEpoch := psm.sessionsWithAllConsumers[epoch]; consumerFoundInEpoch {
		if providerSessionWithConsumer, ProjectIdFound := mapOfProviderSessionsWithConsumer.sessionMap[projectId]; ProjectIdFound {
			if providerSessionWithConsumer.atomicReadConsumerBlocked() == blockListedConsumer { // we atomic read block listed so we dont need to lock the provider. (double lock is always a bad idea.)
				// consumer is blocked.
				utils.LavaFormatWarning("getActiveConsumer", ConsumerIsBlockListed, utils.Attribute{Key: "RequestedEpoch", Value: epoch}, utils.Attribute{Key: "ConsumerAddress", Value: projectId})
				return nil, ConsumerIsBlockListed
			}
			return providerSessionWithConsumer, nil // no error
		}
	}
	return nil, ConsumerNotRegisteredYet
}

func (psm *ProviderSessionManager) getSessionFromAnActiveConsumer(ctx context.Context, providerSessionsWithConsumer *ProviderSessionsWithConsumerProject, sessionId, epoch uint64) (singleProviderSession *SingleProviderSession, err error) {
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
	if epoch <= psm.blockedEpochHeight || epoch <= psm.currentEpoch {
		// this shouldn't happen, but nothing to do
		return
	}
	if epoch > psm.blockDistanceForEpochValidity {
		psm.blockedEpochHeight = epoch - psm.blockDistanceForEpochValidity
	} else {
		psm.blockedEpochHeight = 0
	}
	psm.currentEpoch = epoch
	psm.consumerPairedWithProjectMap = filterOldEpochEntries(psm.blockedEpochHeight, psm.consumerPairedWithProjectMap)
	psm.sessionsWithAllConsumers = filterOldEpochEntries(psm.blockedEpochHeight, psm.sessionsWithAllConsumers)
}

func filterOldEpochEntries[T dataHandler](blockedEpochHeight uint64, allEpochsMap map[uint64]T) (validEpochsMap map[uint64]T) {
	// In order to avoid running over the map twice, (1. mark 2. delete.) better technique is to copy and filter
	// which has better O(n) vs O(2n)
	validEpochsMap = map[uint64]T{}
	for epochStored, value := range allEpochsMap {
		if !IsEpochValidForUse(epochStored, blockedEpochHeight) {
			// epoch is not valid so we don't keep its key in the new map

			value.onDeleteEvent()

			continue
		}
		// if epochStored is ok, copy the value into the new map
		validEpochsMap[epochStored] = value
	}
	return
}

// Called when the reward server has information on a higher cu proof and usage and this providerSessionsManager needs to sync up on it
func (psm *ProviderSessionManager) UpdateSessionCU(consumerAddress string, epoch, sessionID, newCU uint64) error {
	// load the session and update the CU inside

	// Step 1: Lock psm
	getProviderSessionsFromManager := func() (*ProviderSessionsWithConsumerProject, error) {
		psm.lock.RLock()
		defer psm.lock.RUnlock()
		if !psm.IsValidEpoch(epoch) { // checking again because we are now locked and epoch cant change now.
			return nil, utils.LavaFormatError("UpdateSessionCU", InvalidEpochError, utils.Attribute{Key: "RequestedEpoch", Value: epoch})
		}

		providerSessionsWithConsumerMap, ok := psm.sessionsWithAllConsumers[epoch]
		if !ok {
			return nil, utils.LavaFormatError("UpdateSessionCU Failed", EpochIsNotRegisteredError, utils.Attribute{Key: "epoch", Value: epoch})
		}

		projectId, found := psm.readConsumerToPairedWithProjectMap(consumerAddress, epoch)
		if !found {
			return nil, utils.LavaFormatError("UpdateSessionCU Failed readConsumerToPairedWithProjectMap", EpochIsNotRegisteredError, utils.Attribute{Key: "epoch", Value: epoch})
		}

		providerSessionsWithConsumer, foundConsumer := providerSessionsWithConsumerMap.sessionMap[projectId]
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
		rpcProviderEndpoint:           rpcProviderEndpoint,
		blockDistanceForEpochValidity: numberOfBlocksKeptInMemory,
		sessionsWithAllConsumers:      map[uint64]sessionData{},
		consumerPairedWithProjectMap:  map[uint64]*projectConsumerMapping{},
	}
}

func IsEpochValidForUse(targetEpoch, blockedEpochHeight uint64) bool {
	return targetEpoch > blockedEpochHeight
}
