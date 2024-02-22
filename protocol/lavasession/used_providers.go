package lavasession

import (
	"strings"
	"sync"

	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
)

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
	return &UsedProviders{providers: map[string]struct{}{}, unwantedProviders: unwantedProviders, blockOnSyncLoss: map[string]struct{}{}}
}

type UsedProviders struct {
	lock              sync.RWMutex
	providers         map[string]struct{}
	selecting         bool
	unwantedProviders map[string]struct{}
	blockOnSyncLoss   map[string]struct{}
}

func (up *UsedProviders) CurrentlyUsed() int {
	up.lock.RLock()
	defer up.lock.RUnlock()
	return len(up.providers)
}

func (up *UsedProviders) RemoveUsed(provider string, err error) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	if err != nil {
		_, ok := up.blockOnSyncLoss[provider]
		if !ok && IsSessionSyncLoss(err) {
			up.blockOnSyncLoss[provider] = struct{}{}
			utils.LavaFormatWarning("Identified SyncLoss in provider, not removing it from list for another attempt", err, utils.Attribute{Key: "address", Value: provider})
		} else {
			up.SetUnwanted(provider)
		}
	} else {
		// we got a valid response from this provider, no reason to keep using it
		up.SetUnwanted(provider)
	}
	delete(up.providers, provider)
}

func (up *UsedProviders) AddUsed(sessions ConsumerSessionsMap) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	// this is argument nil safe
	for provider := range sessions { // the key for ConsumerSessionsMap is the provider public address
		up.providers[provider] = struct{}{}
	}
	up.selecting = false
}

func (up *UsedProviders) SetUnwanted(provider string) {
	if up == nil {
		return
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	up.unwantedProviders[provider] = struct{}{}
}

func (up *UsedProviders) TryLockSelection() bool {
	if up == nil {
		return true
	}
	up.lock.Lock()
	defer up.lock.Unlock()
	if !up.selecting {
		up.selecting = true
		return true
	}
	return false
}

func (up *UsedProviders) GetSelecting() bool {
	if up == nil {
		return false
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	return up.selecting
}

func (up *UsedProviders) GetUnwantedProvidersToSend() map[string]struct{} {
	if up == nil {
		return map[string]struct{}{}
	}
	up.lock.RLock()
	defer up.lock.RUnlock()
	unwantedProvidersToSend := map[string]struct{}{}
	for provider := range up.providers {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	for provider := range up.unwantedProviders {
		unwantedProvidersToSend[provider] = struct{}{}
	}
	return unwantedProvidersToSend
}
