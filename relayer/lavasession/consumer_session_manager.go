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
	validAddressess       []string        // contains all addressess that are currently valid
	providerBlockList     map[string]bool // contains all currently blocked providers, reseted upon epoch change. (easier to search maps.)
	addedToPurgeAndReport map[string]bool // list of purged providers to report for QoS unavailability. (easier to search maps.)

	// pairingPurge - contains all pairings that are unwanted this epoch, keeps them in memory in order to avoid release.
	// (if a consumer session still uses one of them or we want to report it.)
	pairingPurge map[string]*ConsumerSessionsWithProvider
}

// 1. use epoch in order to update a specific epoch.
// 2. update epoch itself
// 3. move current pairings to previous pairings.
// 4. lock and rewrite pairings.
// take care of the following case: request a deletion of a provider from an old epoch, if the epoch is older return an error or do nothing
// 5. providerBlockList reset
func (cs *ConsumerSessionManager) UpdateAllProviders(ctx context.Context, epoch uint64, pairingList []*ConsumerSessionsWithProvider) error {
	pairingListLength := len(pairingList)

	cs.lock.Lock()         // start by locking the class lock.
	defer cs.lock.Unlock() // we defer here so in case we return an error it will unlock automatically.

	if epoch <= cs.currentEpoch { // sentry shouldnt update an old epoch or current epoch
		return utils.LavaFormatError("trying to update provider list for older epoch", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10)})
	}
	// Update Epoch.
	cs.currentEpoch = epoch // TODO_RAN: switch to atomic write.

	// Reset States
	cs.validAddressess = make([]string, pairingListLength)
	cs.providerBlockList = make(map[string]bool, 0)
	cs.pairingAdressess = make([]string, pairingListLength)
	cs.addedToPurgeAndReport = make(map[string]bool, 0)

	// Reset the pairingPurge.
	// This happens only after an entire epoch. so its impossible to have session connected to the old purged list
	cs.pairingPurge = cs.pairing
	cs.pairing = make(map[string]*ConsumerSessionsWithProvider, pairingListLength)
	for idx, provider := range pairingList {
		cs.pairingAdressess[idx] = provider.Acc
		cs.pairing[provider.Acc] = provider
	}
	copy(cs.validAddressess, cs.pairingAdressess) // the starting point is that valid addressess are equal to pairing addressess.

	return nil
}

// reads cs.currentEpoch atomically
func (cs *ConsumerSessionManager) atomicReadCurrentEpoch() (epoch uint64) {
	return atomic.LoadUint64(&cs.currentEpoch)
}

// GetSession will return a ConsumerSession, given cu needed for that session.
// The user can also request specific providers to not be included in the search for a session.
func (cs *ConsumerSessionManager) GetSession(ctx context.Context, cuNeededForSession uint64, initUnwantedProviders map[string]bool) (
	clinetSession *ConsumerSession, epoch uint64, errRet error) {

	providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith := initUnwantedProviders // list of providers that are ignored when picking a valid provider.
	for {

		// Get a valid consumerSessionWithProvider
		consumerSessionWithProvider, providerAddress, sessionEpoch, err := cs.getValidConsumerSessionsWithProvider(providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith, cuNeededForSession)
		if err != nil {
			if PairingListEmptyError.Is(err) {
				return nil, 0, err
			} else if MaxComputeUnitsExceededError.Is(err) {
				// This provider doesnt have enough compute units for this session, we block it for this session and continue to another provider.
				providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith[providerAddress] = false
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
				err = cs.providerBlock(providerAddress, true, sessionEpoch) // reporting and blocking provider this epoch
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
				providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith[providerAddress] = false
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}

		// If we successfully got a consumerSession we can apply the current CU to the consumerSessionWithProvider.UsedComputeUnits
		err = consumerSessionWithProvider.addUsedComputeUnits(cuNeededForSession)
		if err != nil {
			if MaxComputeUnitsExceededError.Is(err) {
				providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith[providerAddress] = false
				// We must unlock the consumer session before continuing.
				consumerSession.lock.Unlock()
				continue
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}

		// Successfully created/got a consumerSession.
		return consumerSession, sessionEpoch, nil
	}
}

// Get a valid provider address.
func (cs *ConsumerSessionManager) getValidProviderAddress(ignoredProvidersList map[string]bool) (address string, err error) {
	// cs.Lock must be Rlocked here.
	ignoredProvidersListLength := len(ignoredProvidersList)
	validAddressessLength := len(cs.validAddressess)
	totalValidLength := validAddressessLength - ignoredProvidersListLength
	if totalValidLength <= 0 {
		err = sdkerrors.Wrapf(PairingListEmptyError, "lookup - cs.validAddressess is empty")
		return
	}
	validAddressIndex := rand.Intn(totalValidLength) // get the N'th valid provider index, only valid providers will increase the addressIndex counter
	validAddressessCounter := 0                      // this counter will try to reach the addressIndex
	for index := 0; index < validAddressessLength; index++ {
		if _, ok := ignoredProvidersList[cs.validAddressess[index]]; !ok { // not ignored -> yes valid
			validAddressessCounter += 1
		}
		if validAddressessCounter == validAddressIndex {
			return cs.validAddressess[validAddressIndex], nil
		}
	}
	return "", UnreachableCodeError // should not reach here
}

func (cs *ConsumerSessionManager) getValidConsumerSessionsWithProvider(ignoredProvidersList map[string]bool, cuNeededForSession uint64) (consumerSessionWithProvider *ConsumerSessionsWithProvider, providerAddress string, currentEpoch uint64, err error) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	providerAddress, err = cs.getValidProviderAddress(ignoredProvidersList)
	if err != nil {
		return nil, "", 0, utils.LavaFormatError("couldnt get a provider address", err, nil)
	}
	consumerSessionWithProvider = cs.pairing[providerAddress]
	if err := consumerSessionWithProvider.validateComputeUnits(cuNeededForSession); err != nil { // checking if we even have enough compute units for this provider.
		return nil, "", 0, err
	}
	currentEpoch = cs.currentEpoch // reading the epoch here while locked, to get the epoch of the pairing.
	return
}

func (cs *ConsumerSessionManager) removeAddressFromValidAddressess(address string) error {
	if cs.lock.TryLock() {
		// if we managed to lock throw an error for misuse.
		defer cs.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "cs.lock must be locked before accessing this method")
	}

	// cs Must be Locked here.
	for idx, addr := range cs.validAddressess {
		if addr == address {
			// remove the index from the valid list.
			cs.validAddressess = append(cs.validAddressess[:idx], cs.validAddressess[idx+1:]...)
			return nil
		}
	}
	return AddressIndexWasNotFoundError
}

// Blocks a provider making him unavailable for pick this epoch, will also report him as unavailable if reportProvider is set to true.
// validates that the sessionEpoch is equal to cs.currentEpoch otherwise does'nt take effect.
func (cs *ConsumerSessionManager) providerBlock(address string, reportProvider bool, sessionEpoch uint64) error {
	// find Index of the address
	if sessionEpoch != cs.atomicReadCurrentEpoch() { // we read here atomicly so cs.currentEpoch cant change in the middle, so we can save time if epochs mismatch
		return EpochMismatchError
	}

	cs.lock.Lock() // we lock RW here because we need to make sure nothing changes while we verify validAddressess/addedToPurgeAndReport/providerBlockList
	defer cs.lock.Unlock()
	if sessionEpoch != cs.currentEpoch { // After we lock we need to verify again that the epoch didnt change while we waited for the lock.
		return EpochMismatchError
	}

	err := cs.removeAddressFromValidAddressess(address)
	if err != nil {
		return err
	}

	if reportProvider { // Report provider flow
		if _, ok := cs.addedToPurgeAndReport[address]; !ok { // verify it does'nt exist already
			cs.addedToPurgeAndReport[address] = true
		}
	}
	if _, ok := cs.providerBlockList[address]; !ok { // verify it does'nt exist already
		cs.providerBlockList[address] = true
	}

	return nil
}

// Report session failiure, mark it as blocked from future usages, report if timeout happened.
func (cs *ConsumerSessionManager) SessionFailure(consumerSession *ConsumerSession, errorReceived error, didTimeout bool) error {
	// consumerSession must be locked when getting here.
	if consumerSession.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method, additional info:", errorReceived)
	}

	// client Session should be locked here. so we can just apply the session failure here.
	if consumerSession.blocklisted {
		// if client session is already blocklisted return an error.
		return sdkerrors.Wrapf(SessionIsAlreadyBlockListedErrror, "trying to report a session failure of a blocklisted client session", &map[string]string{"consumerSession.blocklisted": strconv.FormatBool(consumerSession.blocklisted)})
	}
	if didTimeout {
		consumerSession.qoSInfo.ConsecutiveTimeOut++
	}
	consumerSession.numberOfFailiures += 1 // increase number of failiures for this session

	// if this session failed more than MaximumNumberOfFailiuresAllowedPerConsumerSession times we block list it.
	if consumerSession.numberOfFailiures > MaximumNumberOfFailiuresAllowedPerConsumerSession {
		consumerSession.blocklisted = true // block this session from future usages
	}
	cuToDecrease := consumerSession.latestRelayCu

	// finished with consumerSession here can unlock.
	consumerSession.lock.Unlock() // we unlock before we change anything in the parent ConsumerSessionsWithProvider

	err := consumerSession.client.decreaseUsedComputeUnits(cuToDecrease) // change the cu in parent
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
		publicProviderAddress, pairingEpoch := consumerSession.client.getPublicLavaAddressAndPairingEpoch()
		err = cs.providerBlock(publicProviderAddress, reportProvider, pairingEpoch)
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
func (cs *ConsumerSessionManager) GetSessionFromAllExcept(ctx context.Context, bannedAddresses map[string]bool, cuNeeded uint64, bannedAddressessEpoch uint64) (clinetSession *ConsumerSession, epoch uint64, err error) {
	// if bannedAddressessEpoch != current epoch, we just return GetSession. locks...
	if bannedAddressessEpoch != cs.atomicReadCurrentEpoch() {
		return cs.GetSession(ctx, cuNeeded, nil)
	} else {
		return cs.GetSession(ctx, cuNeeded, bannedAddresses)
	}
}

// On a successful session this function will update all necessary fields in the consumerSession. and unlock it when it finishes
func (cs *ConsumerSessionManager) DoneWithSession(consumerSession *ConsumerSession, epoch uint64, latestServicedBlock int64) error {
	// release locks, update CU, relaynum etc..
	if consumerSession.lock.TryLock() { // verify consumerSession was locked.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method")
	}

	defer consumerSession.lock.Unlock()
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
func (cs *ConsumerSessionManager) GetReportedProviders(epoch uint64) []string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	if epoch != cs.currentEpoch {
		return []string{} // if epochs are not equal, we will return an empty list.
	}
	keys := make([]string, 0, len(cs.addedToPurgeAndReport))
	for k := range cs.addedToPurgeAndReport {
		keys = append(keys, k)
	}
	return keys
}
