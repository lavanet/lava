package provideroptimizer

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/require"
)

const (
	TEST_AVERAGE_BLOCK_TIME = 10 * time.Second
	TEST_BASE_WORLD_LATENCY = 150 * time.Millisecond
)

func setupProviderOptimizer(maxProvidersCount int) *ProviderOptimizer {
	averageBlockTIme := TEST_AVERAGE_BLOCK_TIME
	baseWorldLatency := TEST_BASE_WORLD_LATENCY
	return NewProviderOptimizer(STRATEGY_BALANCED, averageBlockTIme, baseWorldLatency, 1)
}

type providersGenerator struct {
	providersAddresses []string
}

func (pg *providersGenerator) setupProvidersForTest(count int) *providersGenerator {
	pg.providersAddresses = make([]string, count)
	for i := range pg.providersAddresses {
		pg.providersAddresses[i] = "lava@test_" + strconv.Itoa(i)
	}
	return pg
}

func TestProviderOptimizerSetGet(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(1)
	providerAddress := providersGen.providersAddresses[0]
	for i := 0; i < 100; i++ {
		providerData := ProviderData{SyncBlock: uint64(i)}
		address := providerAddress + strconv.Itoa(i)
		set := providerOptimizer.providersStorage.Set(address, providerData, 1)
		if set == false {
			utils.LavaFormatWarning("set in cache dropped", nil)
		} else {
			utils.LavaFormatDebug("successfully set", utils.Attribute{Key: "entry", Value: address})
		}
	}
	for i := 0; i < 100; i++ {
		address := providerAddress + strconv.Itoa(i)
		providerData := providerOptimizer.getProviderData(address)
		require.Equal(t, uint64(i), providerData.SyncBlock, "failed getting entry %s", address)
	}
}

func TestProviderOptimizerBasic(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)

	requestCU := uint64(10)
	requestBlock := int64(1000)
	pertrubationPercentage := 0.0

	returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[1], TEST_BASE_WORLD_LATENCY*3, true)
	returnedProviders = providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[1]) // we shouldn't pick the wrong provider
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[0], TEST_BASE_WORLD_LATENCY/2, true)
	returnedProviders = providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.Equal(t, providersGen.providersAddresses[0], returnedProviders[0]) // we should pick the best provider
}

func TestProviderOptimizerBasicRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)

	requestCU := uint64(10)
	requestBlock := int64(1000)
	pertrubationPercentage := 0.0
	syncBlock := uint64(requestBlock)

	providerOptimizer.AppendRelayData(providersGen.providersAddresses[1], TEST_BASE_WORLD_LATENCY*3, false, true, requestCU, syncBlock)
	returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[1]) // we shouldn't pick the wrong provider
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[0], TEST_BASE_WORLD_LATENCY/2, false, true, requestCU, syncBlock)
	returnedProviders = providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.Equal(t, providersGen.providersAddresses[0], returnedProviders[0]) // we should pick the best provider
}

func TestProviderOptimizerAvailability(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)

	requestCU := uint64(10)
	requestBlock := int64(1000)
	pertrubationPercentage := 0.0
	skipIndex := rand.Intn(providersCount)
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score
		if i == skipIndex {
			// skip 0
			continue
		}
		providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false)
	}
	returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.Equal(t, providersGen.providersAddresses[skipIndex], returnedProviders[0])
	returnedProviders = providerOptimizer.ChooseProvider(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, providersGen.providersAddresses[skipIndex], returnedProviders[0])
}

func TestProviderOptimizerAvailabilityRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)

	requestCU := uint64(10)
	requestBlock := int64(1000)
	pertrubationPercentage := 0.0
	syncBlock := uint64(requestBlock)
	skipIndex := rand.Intn(providersCount)
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score
		if i == skipIndex {
			// skip one provider
			continue
		}
		providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false, false, requestCU, syncBlock)
	}
	returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.Equal(t, providersGen.providersAddresses[skipIndex], returnedProviders[0])
	returnedProviders = providerOptimizer.ChooseProvider(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, providersGen.providersAddresses[skipIndex], returnedProviders[0])
}

func TestProviderOptimizerAvailabilityBlockError(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)

	requestCU := uint64(10)
	requestBlock := int64(1000)
	pertrubationPercentage := 0.0
	syncBlock := uint64(requestBlock)
	// chosenIndex := rand.Intn(providersCount)
	chosenIndex := 1
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score
		if i == chosenIndex {
			// skip one provider
			providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false, true, requestCU, syncBlock)
			continue
		}
		providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false, true, requestCU, syncBlock-1) // update that he doesn't have the latest requested block
	}
	returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock, pertrubationPercentage)
	require.Equal(t, 1, len(returnedProviders))
	require.Equal(t, providersGen.providersAddresses[chosenIndex], returnedProviders[0])

}
