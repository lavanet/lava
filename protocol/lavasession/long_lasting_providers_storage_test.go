package lavasession

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsProviderLongLasting(t *testing.T) {
	llps := NewLongLastingProvidersStorage()

	// Add a provider
	providerAddress := "provider1"
	llps.AddProvider(providerAddress)

	// Check if the provider is long-lasting
	isLongLasting := llps.IsProviderLongLasting(providerAddress)
	require.True(t, isLongLasting)

	// Remove the provider
	llps.RemoveProvider(providerAddress)

	// Check if the provider is still long-lasting
	isLongLasting = llps.IsProviderLongLasting(providerAddress)
	require.False(t, isLongLasting)
}

func TestConcurrentAccess(t *testing.T) {
	llps := NewLongLastingProvidersStorage()

	// Add and remove providers concurrently
	numProviders := 100
	var wg sync.WaitGroup
	wg.Add(numProviders * 2)
	for i := 0; i < numProviders; i++ {
		providerSpecificWg := sync.WaitGroup{}
		providerSpecificWg.Add(1)
		go func(providerIndex int) {
			defer func() {
				wg.Done()
				providerSpecificWg.Done()
			}()

			providerAddress := "provider" + strconv.Itoa(providerIndex)
			llps.AddProvider(providerAddress)
		}(i)

		go func(providerIndex int) {
			providerSpecificWg.Wait()
			defer wg.Done()
			providerAddress := "provider" + strconv.Itoa(providerIndex)
			llps.RemoveProvider(providerAddress)
		}(i)
	}
	wg.Wait()

	// Check if all providers were added and removed
	for i := 0; i < numProviders; i++ {
		providerAddress := "provider" + strconv.Itoa(i)
		isLongLasting := llps.IsProviderLongLasting(providerAddress)
		require.False(t, isLongLasting)
	}
}
