package lavasession

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
)

const MaximumNumberOfSelectionLockAttempts = 500

func NewUsedProviders(directiveHeaders map[string]string) *UsedProviders {
	unwantedProviders := map[string]struct{}{}
	if len(directiveHeaders) > 0 {
		blockedProviders, ok := directiveHeaders[common.BLOCK_PROVIDERS_ADDRESSES_HEADER_NAME]
		if ok {
			providerAddressesToBlock := strings.Split(blockedProviders, ",")
			for _, providerAddress := range providerAddressesToBlock {
				unwantedProviders[providerAddress] = struct{}{}
			}
		}
	}
	return &UsedProviders{providers: map[string]struct{}{}, unwantedProviders: unwantedProviders, blockOnSyncLoss: map[string]struct{}{}, erroredProviders: map[string]struct{}{}}
}

type UsedProviders struct {
	lock                sync.RWMutex
	providers           map[string]struct{}
	selecting           bool
	unwantedProviders   map[string]struct{}
	erroredProviders    map[string]struct{} // providers who returned protocol errors (used to debug relays for now)
	blockOnSyncLoss     map[string]struct{}
	sessionsLatestBatch int
}

func (up *UsedProviders) CurrentlyUsed() int {
	if up == nil {
		utils.LavaFormatError("UsedProviders.CurrentlyUsed is nil, misuse detected", nil)
		return 0
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	return len(up.providers)
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

func (up *UsedProviders) CurrentlyUsedAddresses() []string {
	if up == nil {
		utils.LavaFormatError("UsedProviders.CurrentlyUsedAddresses is nil, misuse detected", nil)
		return []string{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	addresses := []string{}
	for addr := range up.providers {
		addresses = append(addresses, addr)
	}
	return addresses
}

func (up *UsedProviders) UnwantedAddresses() []string {
	if up == nil {
		utils.LavaFormatError("UsedProviders.UnwantedAddresses is nil, misuse detected", nil)
		return []string{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	addresses := []string{}
	for addr := range up.unwantedProviders {
		addresses = append(addresses, addr)
	}
	return addresses
}

func (up *UsedProviders) AddUnwantedAddresses(address string) {
	if up == nil {
		utils.LavaFormatError("UsedProviders.AddUnwantedAddresses is nil, misuse detected", nil)
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	up.unwantedProviders[address] = struct{}{}
}

func (up *UsedProviders) RemoveUsed(provider string, err error) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	if err != nil {
		up.erroredProviders[provider] = struct{}{}
		if shouldRetryWithThisError(err) {
			_, ok := up.blockOnSyncLoss[provider]
			if !ok && IsSessionSyncLoss(err) {
				up.blockOnSyncLoss[provider] = struct{}{}
				utils.LavaFormatWarning("Identified SyncLoss in provider, allowing retry", err, utils.Attribute{Key: "address", Value: provider})
			} else {
				up.setUnwanted(provider)
			}
		} else {
			up.setUnwanted(provider)
		}
	} else {
		// we got a valid response from this provider, no reason to keep using it
		up.setUnwanted(provider)
	}
	delete(up.providers, provider)
}

func (up *UsedProviders) ClearUnwanted() {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	// this is nil safe
	up.unwantedProviders = map[string]struct{}{}
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
		for provider := range sessions { // the key for ConsumerSessionsMap is the provider public address
			up.providers[provider] = struct{}{}
			up.sessionsLatestBatch++
		}
	}
	up.selecting = false
}

// called when already locked.
func (up *UsedProviders) setUnwanted(provider string) {
	if up == nil {
		return
	}
	up.unwantedProviders[provider] = struct{}{}
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

func (up *UsedProviders) GetErroredProviders() map[string]struct{} {
	if up == nil {
		return map[string]struct{}{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	return up.erroredProviders
}

func (up *UsedProviders) GetUnwantedProvidersToSend() map[string]struct{} {
	if up == nil {
		return map[string]struct{}{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	unwantedProvidersToSend := map[string]struct{}{}
	// block the currently used providers
	for provider := range up.providers {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	// block providers that we have a response for
	for provider := range up.unwantedProviders {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	return unwantedProvidersToSend
}

func shouldRetryWithThisError(err error) bool {
	return IsSessionSyncLoss(err)
}
