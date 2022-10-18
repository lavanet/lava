package lavasession

import (
	"context"
	"strconv"
	"sync"

	"github.com/lavanet/lava/utils"
)

// const EpochMemory =
// save X amount of epochs
type ConsumerSessionsWithProvider struct {
	pairingMu    sync.RWMutex
	pairing      map[string]*RelayerClientWrapper // key == provider addersses
	currentEpoch uint64

	// extra locks for others
	pairingAdressess      []string // contains all addressess
	providerBlockList     []string // reset upon epoch change
	previousEpochPairings map[string]*RelayerClientWrapper
	addedToPurgeAndReport []string // list of purged providers to report.
	pairingPurge          map[string]*RelayerClientWrapper
}

//
func (cs *ConsumerSessionsWithProvider) UpdateAllProviders(ctx context.Context, epoch uint64, pairingList []*RelayerClientWrapper) error {
	if epoch < cs.currentEpoch { // sentry shouldnt update an old epoch
		return utils.LavaFormatError("trying to update provider list for older epoch", nil, &map[string]string{"epoch": strconv.FormatUint(epoch, 10)})
	}

	// 1. use epoch in order to update a specific epoch.
	// 2. update epoch itself
	// 3. move current pairings to previous pairings.
	// 4. lock and rewrite pairings.
	// take care of the following case: request a deletion of a provider from an old epoch, if the epoch is older return an error or do nothing
	// 5. providerBlockList reset

	cs.pairingMu.Lock()
	defer cs.pairingMu.Unlock()
	return nil
}

//
func (cs *ConsumerSessionsWithProvider) GetSession(cuNeededForSession uint64) (clinetSession *ClientSession, epoch uint64, err error) {
	// 0. lock pairing for Read only - dont forget to release upon failiures

	// 1. get a random provider from pairing map
	// 2. make sure he responds
	// 		-> if not try different endpoint ->
	// 			->  if yes, make sure no more than X(10 currently) paralel sessions only when new session is needed validate this
	//          	->  limit the number of blocklisted sessions for each provider?/endpoint? to X (3?) if a provider has more than X, blacklist it.
	// 			-> if all endpoints are dead get another provider
	// 				-> try again
	// 3. after session is picked / created, we lock it and return it

	// design:
	// random select over providers with retry
	// 	   loop over endpoints
	//         loop over sessions (not black listed [with atomic read because its not locked] and can be locked)

	// UsedComputeUnits - updating the provider of the cu that will be used upon getsession success.
	// if session will fail in the future this amount should be deducted

	// check PairingListEmpty error

	return nil, cs.currentEpoch, nil // TODO_RAN: switch cs.currentEpoch to atomic read
}

// report a failure with the provider.
func (cs *ConsumerSessionsWithProvider) ProviderFailure(epoch uint64, address string) error {
	// add providdr address to block list, if epoch is not older than currentEpoch.
	// checks here for anything changed while waiting for lock (epoch / pairing doesnt excisits anymore etc..)
	//
	return nil
}

func (cs *ConsumerSessionsWithProvider) SessionFailure(clientSession *ClientSession) error {
	// try lock?
	// client Session should be locked here. so we can just apply the session failure here.
	if clientSession.blocklisted {
		// if client session is already blocklisted return an error.
		return utils.LavaFormatError("trying to report a session failure of a blocklisted client session", nil, &map[string]string{"clientSession.blocklisted": strconv.FormatBool(clientSession.blocklisted)})
	}
	// 1. if we failed we need to update the session UsedComputeUnits.
	// 2. clientSession.blocklisted = true
	return nil
}

// get a session from a specific provider, pairing must be locked before accessing here.
func (cs *ConsumerSessionsWithProvider) getSessionFromAProvider(address string) (clinetSession *ClientSession, epoch uint64, err error) {
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

func (cs *ConsumerSessionsWithProvider) getEndpointFromProvider(address string) (connected bool, endpointPtr *Endpoint) {
	// get the code from sentry FetchEndpointConnectionFromClientWrapper
	return false, nil
}

// get a session from the pool except a specific providers
func (cs *ConsumerSessionsWithProvider) getSessionFromAllExcept(bannedAddresses []string) (clinetSession *ClientSession, epoch uint64, err error) {
	// iterate over the list of providers and fetch one that is not in the bannedAddress list
	// if all providers are purged and pairing list is empty throw PairingListEmpty error.
	return nil, cs.currentEpoch, nil
}

func (cs *ConsumerSessionsWithProvider) DoneWithSession(epoch uint64, cuUsed uint64) error {
	// release locks, update CU, relaynum etc..
	return nil
}

func (cs *ConsumerSessionsWithProvider) GetBlockedProviders(epoch uint64) (string, error) {
	// Rlock providerBlockList
	// return providerBlockList
	return "", nil
}
