package lavasession

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsProviderInActiveSubscription(t *testing.T) {
	acps := NewActiveSubscriptionProvidersStorage()

	// Add a provider
	providerAddress := "provider1"
	acps.AddProvider(providerAddress)

	// Check if the provider is long-lasting
	isActiveSubscription := acps.IsProviderCurrentlyUsed(providerAddress)
	require.True(t, isActiveSubscription)

	// Remove the provider
	acps.RemoveProvider(providerAddress)

	// Check if the provider is still long-lasting
	isActiveSubscription = acps.IsProviderCurrentlyUsed(providerAddress)
	require.False(t, isActiveSubscription)
}

func TestConcurrentAccess(t *testing.T) {
	acps := NewActiveSubscriptionProvidersStorage()

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
			acps.AddProvider(providerAddress)
		}(i)

		go func(providerIndex int) {
			providerSpecificWg.Wait()
			defer wg.Done()
			providerAddress := "provider" + strconv.Itoa(providerIndex)
			acps.RemoveProvider(providerAddress)
		}(i)
	}
	wg.Wait()

	// Check if all providers were added and removed
	for i := 0; i < numProviders; i++ {
		providerAddress := "provider" + strconv.Itoa(i)
		isCurrentlyActive := acps.IsProviderCurrentlyUsed(providerAddress)
		require.False(t, isCurrentlyActive)
	}
}
