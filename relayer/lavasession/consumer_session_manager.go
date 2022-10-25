package lavasession

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
)

type ConsumerSessionManager struct {
	lock         sync.RWMutex
	pairing      map[string]*ConsumerSessionsWithProvider // key == provider adderss
	currentEpoch uint64

	// pairingAdressess for Data reliability
	pairingAdressess []string // contains all addressess from the initial pairing.
	// providerBlockList + validAddressess == pairingAdressess (while locked)
	validAddressess       []string            // contains all addressess that are currently valid
	providerBlockList     map[string]struct{} // contains all currently blocked providers, reseted upon epoch change. (easier to search maps.)
	addedToPurgeAndReport map[string]struct{} // list of purged providers to report for QoS unavailability. (easier to search maps.)

	// pairingPurge - contains all pairings that are unwanted this epoch, keeps them in memory in order to avoid release.
	// (if a consumer session still uses one of them or we want to report it.)
	pairingPurge map[string]*ConsumerSessionsWithProvider
}

// Update the provider pairing list for the ConsumerSessionManager
func (csm *ConsumerSessionManager) UpdateAllProviders(ctx context.Context, epoch uint64, pairingList []*ConsumerSessionsWithProvider) error {
	pairingListLength := len(pairingList)

	csm.lock.Lock()         // start by locking the class lock.
	defer csm.lock.Unlock() // we defer here so in case we return an error it will unlock automatically.

	if epoch <= csm.atomicReadCurrentEpoch() { // sentry shouldnt update an old epoch or current epoch
		return utils.LavaFormatError("trying to update provider list for older epoch", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "currentEpoch": strconv.FormatUint(csm.atomicReadCurrentEpoch(), 10)})
	}
	// Update Epoch.
	csm.atomicWriteCurrentEpoch(epoch)

	// Reset States
	csm.validAddressess = make([]string, pairingListLength)
	csm.providerBlockList = make(map[string]struct{}, 0)
	csm.pairingAdressess = make([]string, pairingListLength)
	csm.addedToPurgeAndReport = make(map[string]struct{}, 0)

	// Reset the pairingPurge.
	// This happens only after an entire epoch. so its impossible to have session connected to the old purged list
	csm.pairingPurge = csm.pairing
	csm.pairing = make(map[string]*ConsumerSessionsWithProvider, pairingListLength)
	for idx, provider := range pairingList {
		csm.pairingAdressess[idx] = provider.Acc
		csm.pairing[provider.Acc] = provider
	}
	copy(csm.validAddressess, csm.pairingAdressess) // the starting point is that valid addressess are equal to pairing addressess.

	return nil
}

// reads cs.currentEpoch atomically
func (csm *ConsumerSessionManager) atomicWriteCurrentEpoch(epoch uint64) {
	atomic.StoreUint64(&csm.currentEpoch, epoch)
}

// reads cs.currentEpoch atomically
func (csm *ConsumerSessionManager) atomicReadCurrentEpoch() (epoch uint64) {
	return atomic.LoadUint64(&csm.currentEpoch)
}

// GetSession will return a ConsumerSession, given cu needed for that session.
// The user can also request specific providers to not be included in the search for a session.
func (csm *ConsumerSessionManager) GetSession(ctx context.Context, cuNeededForSession uint64, initUnwantedProviders map[string]struct{}) (
	clinetSession *ConsumerSession, epoch uint64, errRet error) {

	// providers that we dont try to connect this iteration.
	tempIgnoredProviders := &ignoredProviders{
		providers:    initUnwantedProviders,
		currentEpoch: csm.atomicReadCurrentEpoch(),
	}
	for {
		// Get a valid consumerSessionWithProvider
		consumerSessionWithProvider, providerAddress, sessionEpoch, err := csm.getValidConsumerSessionsWithProvider(tempIgnoredProviders, cuNeededForSession)
		if err != nil {
			if PairingListEmptyError.Is(err) {
				return nil, 0, err
			} else if MaxComputeUnitsExceededError.Is(err) {
				// This provider doesnt have enough compute units for this session, we block it for this session and continue to another provider.
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}

		// Get a valid Endpoint from the provider chosen
		connected, endpoint, err := consumerSessionWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider(ctx, sessionEpoch)
		if err != nil {
			// verify err is AllProviderEndpointsDisabled and report.
			if AllProviderEndpointsDisabledError.Is(err) {
				err = csm.blockProvider(providerAddress, true, sessionEpoch) // reporting and blocking provider this epoch
				if err != nil {
					if !EpochMismatchError.Is(err) {
						// only acceptable error is EpochMismatchError so if different, throw fatal
						utils.LavaFormatFatal("Unsupported Error", err, nil)
					}
				}
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		} else if !connected {
			// If failed to connect we try again getting a random provider to pick from
			continue
		}

		// Get session from endpoint or create new or continue. if more than 10 connections are open.
		consumerSession, err := consumerSessionWithProvider.getConsumerSessionInstanceFromEndpoint(endpoint)
		if err != nil {
			if MaximumNumberOfSessionsExceededError.Is(err) {
				// we can get a different provider, adding this provider to the list of providers to skip on.
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}

		// If we successfully got a consumerSession we can apply the current CU to the consumerSessionWithProvider.UsedComputeUnits
		err = consumerSessionWithProvider.addUsedComputeUnits(cuNeededForSession)
		if err != nil {
			if MaxComputeUnitsExceededError.Is(err) {
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				// We must unlock the consumer session before continuing.
				consumerSession.lock.Unlock()
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		} else {
			consumerSession.latestRelayCu = cuNeededForSession // set latestRelayCu
			// Successfully created/got a consumerSession.
			return consumerSession, sessionEpoch, nil
		}
		utils.LavaFormatFatal("Unsupported Error", UnreachableCodeError, nil)
	}
}

// Get a valid provider address.
func (csm *ConsumerSessionManager) getValidProviderAddress(ignoredProvidersList map[string]struct{}) (address string, err error) {
	// cs.Lock must be Rlocked here.
	ignoredProvidersListLength := len(ignoredProvidersList)
	validAddressessLength := len(csm.validAddressess)
	totalValidLength := validAddressessLength - ignoredProvidersListLength
	if totalValidLength <= 0 {
		err = PairingListEmptyError
		return
	}
	validAddressIndex := rand.Intn(totalValidLength) // get the N'th valid provider index, only valid providers will increase the addressIndex counter
	validAddressessCounter := 0                      // this counter will try to reach the addressIndex
	for index := 0; index < validAddressessLength; index++ {
		if _, ok := ignoredProvidersList[csm.validAddressess[index]]; !ok { // not ignored -> yes valid
			if validAddressessCounter == validAddressIndex {
				return csm.validAddressess[validAddressIndex], nil
			}
			validAddressessCounter += 1
		}
	}
	return "", UnreachableCodeError // should not reach here
}

func (csm *ConsumerSessionManager) getValidConsumerSessionsWithProvider(ignoredProviders *ignoredProviders, cuNeededForSession uint64) (consumerSessionWithProvider *ConsumerSessionsWithProvider, providerAddress string, currentEpoch uint64, err error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	currentEpoch = csm.atomicReadCurrentEpoch() // reading the epoch here while locked, to get the epoch of the pairing.
	if ignoredProviders.currentEpoch < currentEpoch {
		ignoredProviders.providers = nil // reset the old providers as epochs changed so we have a new pairing list.
		ignoredProviders.currentEpoch = currentEpoch
	}

	providerAddress, err = csm.getValidProviderAddress(ignoredProviders.providers)
	if err != nil {
		utils.LavaFormatError("couldnt get a provider address", err, nil)
		return nil, "", 0, err
	}
	consumerSessionWithProvider = csm.pairing[providerAddress]
	if err := consumerSessionWithProvider.validateComputeUnits(cuNeededForSession); err != nil { // checking if we even have enough compute units for this provider.
		return nil, providerAddress, 0, err // provider address is used to add to temp ignore upon error
	}
	return
}

// removes a given address from the valid addressess list.
func (csm *ConsumerSessionManager) removeAddressFromValidAddressess(address string) error {
	// cs Must be Locked here.
	for idx, addr := range csm.validAddressess {
		if addr == address {
			// remove the index from the valid list.
			csm.validAddressess = append(csm.validAddressess[:idx], csm.validAddressess[idx+1:]...)
			return nil
		}
	}
	return AddressIndexWasNotFoundError
}

// Blocks a provider making him unavailable for pick this epoch, will also report him as unavailable if reportProvider is set to true.
// Validates that the sessionEpoch is equal to cs.currentEpoch otherwise does'nt take effect.
func (csm *ConsumerSessionManager) blockProvider(address string, reportProvider bool, sessionEpoch uint64) error {
	// find Index of the address
	if sessionEpoch != csm.atomicReadCurrentEpoch() { // we read here atomicly so cs.currentEpoch cant change in the middle, so we can save time if epochs mismatch
		return EpochMismatchError
	}

	csm.lock.Lock() // we lock RW here because we need to make sure nothing changes while we verify validAddressess/addedToPurgeAndReport/providerBlockList
	defer csm.lock.Unlock()
	if sessionEpoch != csm.atomicReadCurrentEpoch() { // After we lock we need to verify again that the epoch didnt change while we waited for the lock.
		return EpochMismatchError
	}

	err := csm.removeAddressFromValidAddressess(address)
	if err != nil {
		return err
	}

	if reportProvider { // Report provider flow
		if _, ok := csm.addedToPurgeAndReport[address]; !ok { // verify it does'nt exist already
			csm.addedToPurgeAndReport[address] = struct{}{}
		}
	}
	if _, ok := csm.providerBlockList[address]; !ok { // verify it does'nt exist already
		csm.providerBlockList[address] = struct{}{}
	}
	return nil
}

// Report session failiure, mark it as blocked from future usages, report if timeout happened.
func (csm *ConsumerSessionManager) OnSessionFailure(consumerSession *ConsumerSession, errorReceived error, didTimeout bool) error {
	// consumerSession must be locked when getting here.
	if consumerSession.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method, additional info:")
	}

	// client Session should be locked here. so we can just apply the session failure here.
	if consumerSession.blocklisted {
		// if client session is already blocklisted return an error.
		return sdkerrors.Wrapf(SessionIsAlreadyBlockListedErrror, "trying to report a session failure of a blocklisted client session")
	}
	if didTimeout {
		consumerSession.qoSInfo.ConsecutiveTimeOut++
	}
	consumerSession.numberOfFailiures += 1 // increase number of failiures for this session

	// if this session failed more than MaximumNumberOfFailiuresAllowedPerConsumerSession times we block list it.
	if consumerSession.numberOfFailiures > MaximumNumberOfFailiuresAllowedPerConsumerSession {
		consumerSession.blocklisted = true // block this session from future usages
	} else if SessionOutOfSyncError.Is(errorReceived) { // this is an error that we must block the session due to.
		consumerSession.blocklisted = true
	}
	cuToDecrease := consumerSession.latestRelayCu
	consumerSession.latestRelayCu = 0 // making sure no one uses it in a wrong way

	parentConsumerSessionsWithProvider := consumerSession.client // must read this pointer before unlocking
	// finished with consumerSession here can unlock.
	consumerSession.lock.Unlock() // we unlock before we change anything in the parent ConsumerSessionsWithProvider

	err := parentConsumerSessionsWithProvider.decreaseUsedComputeUnits(cuToDecrease) // change the cu in parent
	if err != nil {
		return err
	}

	// check if need to block & report
	var blockProvider, reportProvider bool
	if ReportAndBlockProviderError.Is(errorReceived) {
		blockProvider = true
		reportProvider = true
	} else if BlockProviderError.Is(errorReceived) {
		blockProvider = true
	}
	if blockProvider {
		publicProviderAddress, pairingEpoch := parentConsumerSessionsWithProvider.getPublicLavaAddressAndPairingEpoch()
		err = csm.blockProvider(publicProviderAddress, reportProvider, pairingEpoch)
		if err != nil {
			if EpochMismatchError.Is(err) {
				return nil // no effects this epoch has been changed
			}
			return err
		}
	}
	return nil
}

// get a session from the pool except specific providers, which also validates the epoch.
func (csm *ConsumerSessionManager) GetSessionFromAllExcept(ctx context.Context, bannedAddresses map[string]struct{}, cuNeeded uint64, bannedAddressessEpoch uint64) (clinetSession *ConsumerSession, epoch uint64, err error) {
	// if bannedAddressessEpoch != current epoch, we just return GetSession. locks...
	if bannedAddressessEpoch != csm.atomicReadCurrentEpoch() {
		return csm.GetSession(ctx, cuNeeded, nil)
	} else {
		return csm.GetSession(ctx, cuNeeded, bannedAddresses)
	}
}

// On a successful session this function will update all necessary fields in the consumerSession. and unlock it when it finishes
func (csm *ConsumerSessionManager) OnSessionDone(consumerSession *ConsumerSession, epoch uint64, latestServicedBlock int64) error {
	// release locks, update CU, relaynum etc..
	defer consumerSession.lock.Unlock() // we neeed to be locked here, if we didnt get it locked we try lock anyway
	if consumerSession.lock.TryLock() { // verify consumerSession was locked.
		// if we managed to lock throw an error for misuse.
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method")
	}

	consumerSession.cuSum += consumerSession.latestRelayCu
	consumerSession.latestRelayCu = 0 // reset cu just in case
	// increase relayNum
	consumerSession.relayNum += 1
	// increase QualityOfService
	consumerSession.qoSInfo.TotalRelays++
	consumerSession.qoSInfo.ConsecutiveTimeOut = 0
	consumerSession.qoSInfo.AnsweredRelays++

	consumerSession.latestBlock = latestServicedBlock
	return nil
}

// Get the reported providers currently stored in the session manager.
func (csm *ConsumerSessionManager) GetReportedProviders(epoch uint64) []string {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	if epoch != csm.atomicReadCurrentEpoch() {
		return []string{} // if epochs are not equal, we will return an empty list.
	}
	keys := make([]string, 0, len(csm.addedToPurgeAndReport))
	for k := range csm.addedToPurgeAndReport {
		keys = append(keys, k)
	}
	return keys
}
