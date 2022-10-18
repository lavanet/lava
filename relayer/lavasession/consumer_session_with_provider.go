package lavasession

import (
	"context"
	"sync"

	"github.com/coniks-sys/coniks-go/crypto/vrf"
	sentry "github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
)

type ConsumerSessionWithProvider struct {
	pairingMu         sync.RWMutex
	pairingNextMu     sync.RWMutex
	pairing           map[string]*sentry.RelayerClientWrapper
	PairingBlockStart int64
	pairingAddresses  []string
	pairingPurgeLock  utils.LavaMutex
	pairingPurge      map[string]*sentry.RelayerClientWrapper
	// addedToPurgeAndReport is using pairingPurgeLock
	addedToPurgeAndReport []string // list of added to purge providers that will be reported by the remaining providers
	pairingNext           map[string]*sentry.RelayerClientWrapper
	pairingNextAddresses  []string
	VrfSkMu               utils.LavaMutex
	VrfSk                 vrf.PrivateKey
}

//
func (cs *ConsumerSessionWithProvider) UpdateAllProviders(ctx context.Context, epoch uint64) {
	// 1. lock and rewrite pairings.
	// 2. use epoch in order to update a specific epoch.
	// take care of the following case: request a deletion of a provider from an old epoch, if the epoch is older return an error or do nothing
}

//
func (cs *ConsumerSessionWithProvider) GetSession(cuNeededForSession uint64) (epoch uint64, err error) {
	// updating the provider of the cu that will be used upon success.
	// return the epoch as well to know in what epoch you used.
	return 0, nil
}

// report a failure with the provider.
func (cs *ConsumerSessionWithProvider) ProviderFailure(epoch uint64, address string) error {
	// cancel this session if there was an issue with the provider.
	return nil
}

func (cs *ConsumerSessionWithProvider) SessionFailure(epoch uint64) error {
	return nil
}

// get a session from a specific provider
func (cs *ConsumerSessionWithProvider) GetSessionFromProvider(epoch uint64, address string) error {
	return nil
}

// get a session from the pool except a specific provider
func (cs *ConsumerSessionWithProvider) GetSessionFromAllExcept(epoch uint64, address string) error {
	return nil
}

func (cs *ConsumerSessionWithProvider) DoneWithSession(epoch uint64, cuUsed uint64) error {
	return nil
}

func (cs *ConsumerSessionWithProvider) GetBlockedProviders(epoch uint64) (string, error) {
	return "", nil
}
