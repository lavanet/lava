package lavasession

import (
	"context"
	"math/rand"
	"strconv"
	"sync"

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
	validAddressess       []string // contains all addressess that are currently valid
	providerBlockList     []string // contains all currently blocked providers, reseted upon epoch change.
	addedToPurgeAndReport []string // list of purged providers to report for QoS unavailability.

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
	cs.providerBlockList = make([]string, 0)
	cs.pairingAdressess = make([]string, pairingListLength)
	cs.addedToPurgeAndReport = make([]string, 0)

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

func (cs *ConsumerSessionManager) atomicReadCurrentEpoch() (epoch uint64) {
	// TODO_RAN: populate
	return 0
}

// 0. lock pairing for Read only - dont forget to release upon failiures
// 1. get a random provider from pairing map
// 2. make sure he responds
// 		-> if not try different endpoint ->
// 			->  if yes, make sure no more than X(10 currently) paralel sessions only when new session is needed validate this
// 			-> if all endpoints are dead get another provider
// 				-> try again
// 3. after session is picked / created, we lock it and return it
// design:
// random select over providers with retry
// 	   loop over endpoints
//         loop over sessions (not black listed [with atomic read because its not locked] and can be locked)
// UsedComputeUnits - updating the provider of the cu that will be used upon getsession success.
// if session will fail in the future this amount should be deducted
func (cs *ConsumerSessionManager) GetSession(ctx context.Context, cuNeededForSession uint64) (clinetSession *ConsumerSession, epoch uint64, errRet error) {

	providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith := make(map[string]bool, 0) // list of providers that are ignored when picking a valid provider.
	for {

		// Get a valid consumerSessionWithProvider
		consumerSessionWithProvider, providerAddress, sessionEpoch, err := cs.getValidConsumerSessionsWithProvider(providersThatAreNotBlockedYetButWeDontWantToGetSessionsWith, cuNeededForSession)
		if err != nil {
			if PairingListEmpty.Is(err) {
				return nil, 0, err
			} else if MaxComputeUnitsExceeded.Is(err) {
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
			if AllProviderEndpointsDisabled.Is(err) {
				cs.providerBlock(providerAddress, true) // reporting and blocking provider this epoch
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
			if MaximumNumberOfSessionsExceeded.Is(err) {
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
			if MaxComputeUnitsExceeded.Is(err) {
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
		err = sdkerrors.Wrapf(PairingListEmpty, "lookup - cs.validAddressess is empty")
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

// report a failure with the provider.
func (cs *ConsumerSessionManager) providerBlock(address string, reportProvider bool) error {
	// read currentEpoch atomic if its the same we need to lock and read again.
	// validate errorReceived, some errors will blocklist some will not. if epoch is not older than currentEpoch.
	// checks here for anything changed while waiting for lock (epoch / pairing doesnt excisits anymore etc..)
	// validate the error

	// validate the provider is not already blocked as two sessions can report same provider at the same time
	return nil
}

func (cs *ConsumerSessionManager) SessionFailure(clientSession *ConsumerSession, errorReceived error) error {
	// clientSession must be locked when getting here. verify.

	// client Session should be locked here. so we can just apply the session failure here.
	if clientSession.blocklisted {
		// if client session is already blocklisted return an error.
		return utils.LavaFormatError("trying to report a session failure of a blocklisted client session", nil, &map[string]string{"clientSession.blocklisted": strconv.FormatBool(clientSession.blocklisted)})
	}
	// 1. if we failed we need to update the session UsedComputeUnits. -> lock RelayerClientWrapper to modify it
	// 2. clientSession.blocklisted = true
	// 3. report provider if needed. check cases.
	// unlock clientSession.
	return nil
}

// get a session from a specific provider, pairing must be locked before accessing here.
func (cs *ConsumerSessionManager) getSessionFromAProvider(address string, cuNeeded uint64) (clinetSession *ConsumerSession, epoch uint64, err error) {
	// get session inner function, to get a session from a specific provider.
	// get address from -> pairing map[string]*RelayerClientWrapper ->
	// choose a endpoint for the provider: similar to findPairing.
	// for _, session := range wrap.Sessions {
	// 	if session.Endpoint != endpoint {
	// 		//skip sessions that don't belong to the active connection
	// 		continue
	// 	}
	// 	if session.Lock.TryLock() {
	// 		return session
	// 	}
	// }

	return nil, cs.currentEpoch, nil // TODO_RAN: switch cs.currentEpoch to atomic read
}

// get a session from the pool except a specific providers
func (cs *ConsumerSessionManager) GetSessionFromAllExcept(bannedAddresses []string, cuNeeded uint64, bannedAddressessEpoch uint64) (clinetSession *ConsumerSession, epoch uint64, err error) {
	// if bannedAddressessEpoch != current epoch, we just return GetSession. locks...

	// similar to GetSession code. (they should have same inner function)
	return nil, cs.currentEpoch, nil
}

func (cs *ConsumerSessionManager) DoneWithSession(epoch uint64, qosInfo QoSInfo, latestServicedBlock uint64) error {
	// release locks, update CU, relaynum etc..
	// apply LatestRelayCu to CuSum and reset
	// apply QoS
	// apply RelayNum + 1
	// update serviced node LatestBlock (ETH, etc.. ) <- latestServicedBlock
	// unlock clientSession.
	return nil
}

func (cs *ConsumerSessionManager) GetReportedProviders(epoch uint64) (string, error) {
	// Rlock providerBlockList
	// return providerBlockList
	return "", nil
}
