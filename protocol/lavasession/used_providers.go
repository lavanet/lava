package lavasession

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v4/utils"
)

const MaximumNumberOfSelectionLockAttempts = 500

type BlockedProvidersInf interface {
	GetBlockedProviders() []string
}

func NewUsedProviders(blockedProviders BlockedProvidersInf) *UsedProviders {
	unwantedProviders := map[string]struct{}{}
	originalUnwantedProviders := map[string]struct{}{} // we need a new map as map changes are changed by pointer
	if blockedProviders != nil {
		providerAddressesToBlock := blockedProviders.GetBlockedProviders()
		if len(providerAddressesToBlock) > 0 {
			for _, providerAddress := range providerAddressesToBlock {
				unwantedProviders[providerAddress] = struct{}{}
				originalUnwantedProviders[providerAddress] = struct{}{}
			}
		}
	}

	return &UsedProviders{
		uniqueUsedProviders: map[string]*UniqueUsedProviders{GetEmptyRouterKey().String(): {
			providers:         map[string]struct{}{},
			unwantedProviders: unwantedProviders,
			blockOnSyncLoss:   map[string]struct{}{},
			erroredProviders:  map[string]struct{}{},
		}},
		// we keep the original unwanted providers so when we create more unique used providers
		// we can reuse it as its the user's instructions.
		originalUnwantedProviders: originalUnwantedProviders,
	}
}

// unique used providers are specific for an extension router key.
// meaning each extension router key has a different used providers struct
type UniqueUsedProviders struct {
	providers         map[string]struct{}
	unwantedProviders map[string]struct{}
	erroredProviders  map[string]struct{} // providers who returned protocol errors (used to debug relays for now)
	blockOnSyncLoss   map[string]struct{}
}

type UsedProviders struct {
	lock                      sync.RWMutex
	uniqueUsedProviders       map[string]*UniqueUsedProviders
	originalUnwantedProviders map[string]struct{}
	selecting                 bool
	sessionsLatestBatch       int
	batchNumber               int
}

func (up *UsedProviders) CurrentlyUsed() int {
	if up == nil {
		utils.LavaFormatError("UsedProviders.CurrentlyUsed is nil, misuse detected", nil)
		return 0
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	currentlyUsed := 0
	for _, uniqueUsedProviders := range up.uniqueUsedProviders {
		currentlyUsed += len(uniqueUsedProviders.providers)
	}
	return currentlyUsed
}

func (up *UsedProviders) SessionsLatestBatch() int {
	if up == nil {
		utils.LavaFormatError("UsedProviders.SessionsLatestBatch is nil, misuse detected", nil)
		return 0
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	return up.sessionsLatestBatch
}

func (up *UsedProviders) BatchNumber() int {
	if up == nil {
		utils.LavaFormatError("UsedProviders.BatchNumber is nil, misuse detected", nil)
		return 0
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	return up.batchNumber
}

func (up *UsedProviders) CurrentlyUsedAddresses() []string {
	if up == nil {
		utils.LavaFormatError("UsedProviders.CurrentlyUsedAddresses is nil, misuse detected", nil)
		return []string{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	addresses := []string{}
	for _, uniqueUsedProviders := range up.uniqueUsedProviders {
		for addr := range uniqueUsedProviders.providers {
			addresses = append(addresses, addr)
		}
	}
	return addresses
}

func (up *UsedProviders) AllUnwantedAddresses() []string {
	if up == nil {
		utils.LavaFormatError("UsedProviders.UnwantedAddresses is nil, misuse detected", nil)
		return []string{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	addresses := []string{}
	for _, uniqueUsedProviders := range up.uniqueUsedProviders {
		for addr := range uniqueUsedProviders.unwantedProviders {
			addresses = append(addresses, addr)
		}
	}
	return addresses
}

// Use when locked. Checking wether a router key exists in unique used providers,
// if it does, return it. If it doesn't
// creating a new instance and returning it.
func (up *UsedProviders) createOrUseUniqueUsedProvidersForKey(key RouterKey) *UniqueUsedProviders {
	keyString := key.String()
	uniqueUsedProviders, ok := up.uniqueUsedProviders[keyString]
	if !ok {
		uniqueUsedProviders = &UniqueUsedProviders{
			providers:         map[string]struct{}{},
			unwantedProviders: up.originalUnwantedProviders,
			blockOnSyncLoss:   map[string]struct{}{},
			erroredProviders:  map[string]struct{}{},
		}
		up.uniqueUsedProviders[keyString] = uniqueUsedProviders
	}
	return uniqueUsedProviders
}

func (up *UsedProviders) AddUnwantedAddresses(address string, routerKey RouterKey) {
	if up == nil {
		utils.LavaFormatError("UsedProviders.AddUnwantedAddresses is nil, misuse detected", nil)
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	uniqueUsedProviders := up.createOrUseUniqueUsedProvidersForKey(routerKey)
	uniqueUsedProviders.unwantedProviders[address] = struct{}{}
}

func (up *UsedProviders) RemoveUsed(provider string, routerKey RouterKey, err error) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	uniqueUsedProviders := up.createOrUseUniqueUsedProvidersForKey(routerKey)

	if err != nil {
		uniqueUsedProviders.erroredProviders[provider] = struct{}{}
		if shouldRetryWithThisError(err) {
			_, ok := uniqueUsedProviders.blockOnSyncLoss[provider]
			if !ok && IsSessionSyncLoss(err) {
				uniqueUsedProviders.blockOnSyncLoss[provider] = struct{}{}
				utils.LavaFormatWarning("Identified SyncLoss in provider, allowing retry", err, utils.Attribute{Key: "address", Value: provider})
			} else {
				up.setUnwanted(uniqueUsedProviders, provider)
			}
		} else {
			up.setUnwanted(uniqueUsedProviders, provider)
		}
	} else {
		// we got a valid response from this provider, no reason to keep using it
		up.setUnwanted(uniqueUsedProviders, provider)
	}
	delete(uniqueUsedProviders.providers, provider)
}

func (up *UsedProviders) ClearUnwanted() {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	// this is nil safe
	for _, uniqueUsedProviders := range up.uniqueUsedProviders {
		uniqueUsedProviders.unwantedProviders = map[string]struct{}{}
	}
}

func (up *UsedProviders) AddUsed(sessions ConsumerSessionsMap, err error) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	// this is nil safe
	if len(sessions) > 0 && err == nil {
		up.sessionsLatestBatch = 0
		for provider, sessionInfo := range sessions { // the key for ConsumerSessionsMap is the provider public address
			var routerKey RouterKey
			if sessionInfo.Session != nil {
				routerKey = sessionInfo.Session.routerKey
			} else {
				routerKey = NewRouterKey(nil)
			}
			uniqueUsedProviders := up.createOrUseUniqueUsedProvidersForKey(routerKey)
			uniqueUsedProviders.providers[provider] = struct{}{}
			up.sessionsLatestBatch++
		}
		// increase batch number
		up.batchNumber++
	}
	up.selecting = false
}

// called when already locked.
func (up *UsedProviders) setUnwanted(uniqueUsedProviders *UniqueUsedProviders, provider string) {
	uniqueUsedProviders.unwantedProviders[provider] = struct{}{}
}

func (up *UsedProviders) TryLockSelection(ctx context.Context) error {
	if up == nil {
		return nil
	}
	for counter := 0; counter < MaximumNumberOfSelectionLockAttempts; counter++ {
		select {
		case <-ctx.Done():
			utils.LavaFormatTrace("Failed locking selection, context is done")
			return ContextDoneNoNeedToLockSelectionError
		default:
			canSelect := up.tryLockSelection()
			if canSelect {
				return nil
			}
			time.Sleep(5 * time.Millisecond)
		}
	}

	// if we got here we failed locking the selection.
	return utils.LavaFormatError("Failed locking selection after MaximumNumberOfSelectionLockAttempts", nil, utils.LogAttr("GUID", ctx))
}

func (up *UsedProviders) tryLockSelection() bool {
	up.lock.Lock()
	defer up.lock.Unlock()
	if !up.selecting {
		up.selecting = true
		return true
	}
	return false
}

func (up *UsedProviders) GetErroredProviders(routerKey RouterKey) map[string]struct{} {
	if up == nil {
		return map[string]struct{}{}
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	uniqueUsedProviders := up.createOrUseUniqueUsedProvidersForKey(routerKey)
	return uniqueUsedProviders.erroredProviders
}

func (up *UsedProviders) GetUnwantedProvidersToSend(routerKey RouterKey) map[string]struct{} {
	if up == nil {
		return map[string]struct{}{}
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	uniqueUsedProviders := up.createOrUseUniqueUsedProvidersForKey(routerKey)
	unwantedProvidersToSend := map[string]struct{}{}
	// block the currently used providers
	for provider := range uniqueUsedProviders.providers {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	// block providers that we have a response for
	for provider := range uniqueUsedProviders.unwantedProviders {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	return unwantedProvidersToSend
}

func shouldRetryWithThisError(err error) bool {
	return IsSessionSyncLoss(err)
}
