package lavasession

import (
	"context"
	"encoding/json"
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

	// pairingAddresses for Data reliability
	pairingAddresses       []string // contains all addresses from the initial pairing.
	pairingAddressesLength uint64
	// providerBlockList + validAddresses == pairingAddresses (while locked)
	validAddresses        []string            // contains all addresses that are currently valid
	providerBlockList     map[string]struct{} // contains all currently blocked providers, reset upon epoch change. (easier to search maps.)
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

	if epoch <= csm.atomicReadCurrentEpoch() { // sentry shouldn't update an old epoch or current epoch
		return utils.LavaFormatError("trying to update provider list for older epoch", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10), "currentEpoch": strconv.FormatUint(csm.atomicReadCurrentEpoch(), 10)})
	}
	// Update Epoch.
	csm.atomicWriteCurrentEpoch(epoch)

	// Reset States
	csm.validAddresses = make([]string, pairingListLength)
	csm.providerBlockList = make(map[string]struct{}, 0)
	csm.pairingAddresses = make([]string, pairingListLength)
	csm.addedToPurgeAndReport = make(map[string]struct{}, 0)
	csm.pairingAddressesLength = uint64(pairingListLength)

	// Reset the pairingPurge.
	// This happens only after an entire epoch. so its impossible to have session connected to the old purged list
	csm.pairingPurge = csm.pairing
	csm.pairing = make(map[string]*ConsumerSessionsWithProvider, pairingListLength)
	for idx, provider := range pairingList {
		csm.pairingAddresses[idx] = provider.Acc
		csm.pairing[provider.Acc] = provider
	}
	copy(csm.validAddresses, csm.pairingAddresses) // the starting point is that valid addresses are equal to pairing addresses.

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
	consumerSession *SingleConsumerSession, epoch uint64, providerPublicAddress string, reportedProviders []byte, errRet error) {

	// providers that we don't try to connect this iteration.
	tempIgnoredProviders := &ignoredProviders{
		providers:    initUnwantedProviders,
		currentEpoch: csm.atomicReadCurrentEpoch(),
	}
	for {
		// Get a valid consumerSessionWithProvider
		consumerSessionWithProvider, providerAddress, sessionEpoch, err := csm.getValidConsumerSessionsWithProvider(tempIgnoredProviders, cuNeededForSession)
		if err != nil {
			if PairingListEmptyError.Is(err) {
				return nil, 0, "", nil, err
			} else if MaxComputeUnitsExceededError.Is(err) {
				// This provider does'nt have enough compute units for this session, we block it for this session and continue to another provider.
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

		// we get the reported providers here after we try to connect, so if any provider did'nt respond he will already be added to the list.
		reportedProviders, err = csm.GetReportedProviders(csm.atomicReadCurrentEpoch())
		if err != nil {
			return nil, 0, "", nil, utils.LavaFormatError("Failed Unmarshal Error", err, nil)
		}

		// Get session from endpoint or create new or continue. if more than 10 connections are open.
		consumerSession, pairingEpoch, err := consumerSessionWithProvider.getConsumerSessionInstanceFromEndpoint(endpoint)
		if err != nil {
			if MaximumNumberOfSessionsExceededError.Is(err) {
				// we can get a different provider, adding this provider to the list of providers to skip on.
				tempIgnoredProviders.providers[providerAddress] = struct{}{}
				continue
			} else if MaximumNumberOfBlockListedSessionsError.Is(err) {
				// provider has too many block listed sessions. we block it until the next epoch.
				csm.blockProvider(providerAddress, false, sessionEpoch)
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}

		if pairingEpoch != sessionEpoch {
			// pairingEpoch and SessionEpoch must be the same, we validate them here if they are different we raise an error and continue with pairingEpoch
			utils.LavaFormatError("sessionEpoch and pairingEpoch mismatch", nil, &map[string]string{"sessionEpoch": strconv.FormatUint(sessionEpoch, 10), "pairingEpoch": strconv.FormatUint(pairingEpoch, 10)})
			sessionEpoch = pairingEpoch
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
			consumerSession.LatestRelayCu = cuNeededForSession // set latestRelayCu
			// Successfully created/got a consumerSession.
			return consumerSession, sessionEpoch, providerAddress, reportedProviders, nil
		}
		utils.LavaFormatFatal("Unsupported Error", UnreachableCodeError, nil)
	}
}

// Get a valid provider address.
func (csm *ConsumerSessionManager) getValidProviderAddress(ignoredProvidersList map[string]struct{}) (address string, err error) {
	// cs.Lock must be Rlocked here.
	ignoredProvidersListLength := len(ignoredProvidersList)
	validAddressesLength := len(csm.validAddresses)
	totalValidLength := validAddressesLength - ignoredProvidersListLength
	if totalValidLength <= 0 {
		err = PairingListEmptyError
		return
	}
	validAddressIndex := rand.Intn(totalValidLength) // get the N'th valid provider index, only valid providers will increase the addressIndex counter
	validAddressesCounter := 0                       // this counter will try to reach the addressIndex
	for index := 0; index < validAddressesLength; index++ {
		if _, ok := ignoredProvidersList[csm.validAddresses[index]]; !ok { // not ignored -> yes valid
			if validAddressesCounter == validAddressIndex {
				return csm.validAddresses[validAddressIndex], nil
			}
			validAddressesCounter += 1
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
		utils.LavaFormatError("could'nt get a provider address", err, nil)
		return nil, "", 0, err
	}
	consumerSessionWithProvider = csm.pairing[providerAddress]
	if err := consumerSessionWithProvider.validateComputeUnits(cuNeededForSession); err != nil { // checking if we even have enough compute units for this provider.
		return nil, providerAddress, 0, err // provider address is used to add to temp ignore upon error
	}
	return
}

// removes a given address from the valid addresses list.
func (csm *ConsumerSessionManager) removeAddressFromValidAddresses(address string) error {
	// cs Must be Locked here.
	for idx, addr := range csm.validAddresses {
		if addr == address {
			// remove the index from the valid list.
			csm.validAddresses = append(csm.validAddresses[:idx], csm.validAddresses[idx+1:]...)
			return nil
		}
	}
	return AddressIndexWasNotFoundError
}

// Blocks a provider making him unavailable for pick this epoch, will also report him as unavailable if reportProvider is set to true.
// Validates that the sessionEpoch is equal to cs.currentEpoch otherwise does'nt take effect.
func (csm *ConsumerSessionManager) blockProvider(address string, reportProvider bool, sessionEpoch uint64) error {
	// find Index of the address
	if sessionEpoch != csm.atomicReadCurrentEpoch() { // we read here atomically so cs.currentEpoch cant change in the middle, so we can save time if epochs mismatch
		return EpochMismatchError
	}

	csm.lock.Lock() // we lock RW here because we need to make sure nothing changes while we verify validAddresses/addedToPurgeAndReport/providerBlockList
	defer csm.lock.Unlock()
	if sessionEpoch != csm.atomicReadCurrentEpoch() { // After we lock we need to verify again that the epoch did'nt change while we waited for the lock.
		return EpochMismatchError
	}

	err := csm.removeAddressFromValidAddresses(address)
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

// Report session failure, mark it as blocked from future usages, report if timeout happened.
func (csm *ConsumerSessionManager) OnSessionFailure(consumerSession *SingleConsumerSession, errorReceived error) error {
	// consumerSession must be locked when getting here.
	if consumerSession.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method, additional info:")
	}

	// consumer Session should be locked here. so we can just apply the session failure here.
	if consumerSession.Blocklisted {
		// if consumer session is already blocklisted return an error.
		return sdkerrors.Wrapf(SessionIsAlreadyBlockListedError, "trying to report a session failure of a blocklisted consumer session")
	}

	consumerSession.QoSInfo.TotalRelays++
	consumerSession.ConsecutiveNumberOfFailures += 1 // increase number of failures for this session

	// if this session failed more than MaximumNumberOfFailuresAllowedPerConsumerSession times we block list it.
	if consumerSession.ConsecutiveNumberOfFailures > MaximumNumberOfFailuresAllowedPerConsumerSession {
		consumerSession.Blocklisted = true // block this session from future usages
	} else if SessionOutOfSyncError.Is(errorReceived) { // this is an error that we must block the session due to.
		consumerSession.Blocklisted = true
	}
	cuToDecrease := consumerSession.LatestRelayCu
	consumerSession.LatestRelayCu = 0 // making sure no one uses it in a wrong way

	parentConsumerSessionsWithProvider := consumerSession.Client // must read this pointer before unlocking
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
func (csm *ConsumerSessionManager) GetSessionFromAllExcept(ctx context.Context, bannedAddresses map[string]struct{}, cuNeeded uint64, bannedAddressesEpoch uint64) (consumerSession *SingleConsumerSession, epoch uint64, providerPublicAddress string, reportedProviders []byte, err error) {
	// if bannedAddressesEpoch != current epoch, we just return GetSession. locks...
	if bannedAddressesEpoch != csm.atomicReadCurrentEpoch() {
		return csm.GetSession(ctx, cuNeeded, nil)
	} else {
		return csm.GetSession(ctx, cuNeeded, bannedAddresses)
	}
}

// On a successful session this function will update all necessary fields in the consumerSession. and unlock it when it finishes
func (csm *ConsumerSessionManager) OnSessionDone(consumerSession *SingleConsumerSession, epoch uint64, latestServicedBlock int64) error {
	// release locks, update CU, relaynum etc..
	defer consumerSession.lock.Unlock() // we need to be locked here, if we didn't get it locked we try lock anyway
	if consumerSession.lock.TryLock() { // verify consumerSession was locked.
		// if we managed to lock throw an error for misuse.
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method")
	}

	consumerSession.CuSum += consumerSession.LatestRelayCu
	consumerSession.LatestRelayCu = 0 // reset cu just in case
	// increase relayNum
	consumerSession.RelayNum += RelayNumberIncrement
	// increase QualityOfService
	consumerSession.QoSInfo.TotalRelays++
	consumerSession.ConsecutiveNumberOfFailures = 0
	consumerSession.QoSInfo.AnsweredRelays++

	consumerSession.LatestBlock = latestServicedBlock
	return nil
}

// Get the reported providers currently stored in the session manager.
func (csm *ConsumerSessionManager) GetReportedProviders(epoch uint64) ([]byte, error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	if epoch != csm.atomicReadCurrentEpoch() {
		return []byte{}, nil // if epochs are not equal, we will return an empty list.
	}
	keys := make([]string, 0, len(csm.addedToPurgeAndReport))
	for k := range csm.addedToPurgeAndReport {
		keys = append(keys, k)
	}
	bytes, err := json.Marshal(keys)

	return bytes, err
}

// Data Reliability Section:

// Atomically read csm.pairingAddressesLength for data reliability.
func (csm *ConsumerSessionManager) GetAtomicPairingAddressesLength() uint64 {
	return atomic.LoadUint64(&csm.pairingAddressesLength)
}

func (csm *ConsumerSessionManager) getDataReliabilityProviderIndex(unAllowedAddress string, index uint64) (cswp *ConsumerSessionsWithProvider, providerAddress string, epoch uint64, err error) {
	csm.lock.RLock()
	defer csm.lock.RUnlock()
	currentEpoch := csm.atomicReadCurrentEpoch()
	pairingAddressesLength := csm.GetAtomicPairingAddressesLength()
	if index >= pairingAddressesLength {
		return nil, "", currentEpoch, DataReliabilityIndexOutOfRangeError
	}
	providerAddress = csm.pairingAddresses[index]
	if providerAddress == unAllowedAddress {
		return nil, "", currentEpoch, DataReliabilityIndexRequestedIsOriginalProviderError
	}
	// if address is valid return the ConsumerSessionsWithProvider
	return csm.pairing[providerAddress], providerAddress, currentEpoch, nil

}

func (csm *ConsumerSessionManager) getEndpointFromConsumerSessionWithProviderForDR(ctx context.Context, consumerSessionWithProvider *ConsumerSessionsWithProvider, sessionEpoch uint64, providerAddress string) (endpoint *Endpoint, err error) {
	var connected bool
	for idx := 0; idx < MaxConsecutiveConnectionAttempts; idx++ { // try to connect to the endpoint 3 times
		connected, endpoint, err = consumerSessionWithProvider.fetchEndpointConnectionFromConsumerSessionWithProvider(ctx, sessionEpoch)
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
				break // all endpoints are disabled, no reason to continue with this provider.
			} else {
				utils.LavaFormatFatal("Unsupported Error", err, nil)
			}
		}
		if connected {
			// if we are connected we can stop trying and return the endpoint
			break
		} else {
			continue
		}
	}
	if !connected { // if we are not connected at the end
		// failed to get an endpoint connection from that provider. return an error.
		return nil, FailedToConnectToEndPointForDataReliabilityError
	}
	return endpoint, nil
}

// Get a Data Reliability Session
func (csm *ConsumerSessionManager) GetDataReliabilitySession(ctx context.Context, originalProviderAddress string, index int64) (singleConsumerSession *SingleConsumerSession, providerAddress string, epoch uint64, err error) {
	consumerSessionWithProvider, providerAddress, currentEpoch, err := csm.getDataReliabilityProviderIndex(originalProviderAddress, uint64(index))
	if err != nil {
		return nil, "", 0, err
	}

	// after choosing a provider, try to see if it already has an existing data reliability session.
	err = consumerSessionWithProvider.verifyDataReliabilitySessionWasNotAlreadyCreated()
	if err != nil {
		return nil, "", currentEpoch, err
	}

	// We can get an endpoint now and create a data reliability session.
	endpoint, err := csm.getEndpointFromConsumerSessionWithProviderForDR(ctx, consumerSessionWithProvider, currentEpoch, providerAddress)
	if err != nil {
		return nil, "", currentEpoch, err
	}

	// get data reliability session from endpoint
	consumerSession, pairingEpoch := consumerSessionWithProvider.getDataReliabilitySingleConsumerSession(endpoint)

	if currentEpoch != pairingEpoch { // validate they are the same, if not print an error and set currentEpoch to pairingEpoch.
		utils.LavaFormatError("currentEpoch and pairingEpoch mismatch", nil, &map[string]string{"sessionEpoch": strconv.FormatUint(currentEpoch, 10), "pairingEpoch": strconv.FormatUint(pairingEpoch, 10)})
		currentEpoch = pairingEpoch
	}

	return consumerSession, providerAddress, currentEpoch, nil
}

// On a successful DataReliability session we don't need to increase and update any field, we just need to unlock the session.
func (csm *ConsumerSessionManager) OnDataReliabilitySessionDone(consumerSession *SingleConsumerSession) error {
	defer consumerSession.lock.Unlock() // we need to be locked here, if we didn't get it locked we try lock anyway
	if consumerSession.lock.TryLock() { // verify consumerSession was locked.
		// if we managed to lock throw an error for misuse.
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method")
	}
	return nil
}

// On a failed DataReliability session we don't decrease the cu unlike a normal session, we just unlock and verify if we need to block this session or provider.
func (csm *ConsumerSessionManager) OnDataReliabilitySessionFailure(consumerSession *SingleConsumerSession, errorReceived error) error {
	// consumerSession must be locked when getting here.
	if consumerSession.lock.TryLock() { // verify.
		// if we managed to lock throw an error for misuse.
		defer consumerSession.lock.Unlock()
		return sdkerrors.Wrapf(LockMisUseDetectedError, "consumerSession.lock must be locked before accessing this method, additional info:")
	}
	// consumer Session should be locked here. so we can just apply the session failure here.
	if consumerSession.Blocklisted {
		// if consumer session is already blocklisted return an error.
		return sdkerrors.Wrapf(SessionIsAlreadyBlockListedError, "trying to report a session failure of a blocklisted client session")
	}

	consumerSession.ConsecutiveNumberOfFailures += 1 // increase number of failures for this session
	// if this session failed more than MaximumNumberOfFailuresAllowedPerConsumerSession times we block list it.
	if consumerSession.ConsecutiveNumberOfFailures > MaximumNumberOfFailuresAllowedPerConsumerSession {
		consumerSession.Blocklisted = true // block this session from future usages
	}

	var blockProvider, reportProvider bool
	if ReportAndBlockProviderError.Is(errorReceived) {
		blockProvider = true
		reportProvider = true
	} else if BlockProviderError.Is(errorReceived) {
		blockProvider = true
	}

	parentConsumerSessionsWithProvider := consumerSession.Client
	consumerSession.lock.Unlock()

	if blockProvider {
		publicProviderAddress, pairingEpoch := parentConsumerSessionsWithProvider.getPublicLavaAddressAndPairingEpoch()
		err := csm.blockProvider(publicProviderAddress, reportProvider, pairingEpoch)
		if err != nil {
			if EpochMismatchError.Is(err) {
				return nil // no effects this epoch has been changed
			}
			return err
		}
	}

	return nil
}
