package updaters

import (
	"math"
	"sort"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/stretchr/testify/require"
)

func TestCreateProviderSessionsFromConfig_WithStaticProviders(t *testing.T) {
	tests := []struct {
		name               string
		providers          []*lavasession.RPCStaticProviderEndpoint
		providerNamePrefix string
		epoch              uint64
		expectedCount      int
		expectedNames      []string
	}{
		{
			name: "single static provider",
			providers: []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls: []common.NodeUrl{
						{Url: "https://eth1.example.com"},
					},
					Geolocation: 1,
					Name:        "Ethereum-Primary",
				},
			},
			providerNamePrefix: "StaticProvider_",
			epoch:              100,
			expectedCount:      1,
			expectedNames:      []string{"Ethereum-Primary"},
		},
		{
			name: "multiple static providers",
			providers: []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls: []common.NodeUrl{
						{Url: "https://primary.example.com"},
					},
					Geolocation: 1,
					Name:        "Primary-Ethereum",
				},
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls: []common.NodeUrl{
						{Url: "https://secondary.example.com"},
					},
					Geolocation: 1,
					Name:        "Secondary-Ethereum",
				},
			},
			providerNamePrefix: "StaticProvider_",
			epoch:              200,
			expectedCount:      2,
			expectedNames:      []string{"Primary-Ethereum", "Secondary-Ethereum"},
		},
		{
			name: "provider with multiple endpoints",
			providers: []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls: []common.NodeUrl{
						{Url: "https://node1.example.com"},
						{Url: "https://node2.example.com"},
						{Url: "wss://ws.example.com"},
					},
					Geolocation: 1,
					Name:        "Multi-Endpoint-Provider",
				},
			},
			providerNamePrefix: "StaticProvider_",
			epoch:              300,
			expectedCount:      1,
			expectedNames:      []string{"Multi-Endpoint-Provider"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal PairingUpdater for testing
			pu := &PairingUpdater{}

			// Create a mock RPC endpoint for the function call
			rpcEndpoint := lavasession.RPCEndpoint{
				ChainID:      "ETH1",
				ApiInterface: "jsonrpc",
			}

			existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

			sessions := pu.createProviderSessionsFromConfig(tt.providers, rpcEndpoint, tt.epoch, existingProviders, tt.providerNamePrefix)

			require.Len(t, sessions, tt.expectedCount)

			// Convert map to slice for easier testing
			sessionSlice := make([]*lavasession.ConsumerSessionsWithProvider, 0, len(sessions))
			for _, session := range sessions {
				sessionSlice = append(sessionSlice, session)
			}

			// Sort by PublicLavaAddress to ensure deterministic order for comparison
			sort.Slice(sessionSlice, func(i, j int) bool {
				return sessionSlice[i].PublicLavaAddress < sessionSlice[j].PublicLavaAddress
			})

			// Sort expected names to match the sorted session order
			expectedNamesSorted := make([]string, len(tt.expectedNames))
			copy(expectedNamesSorted, tt.expectedNames)
			sort.Strings(expectedNamesSorted)

			// Create a map from provider names to their config for easier lookup
			providerMap := make(map[string]*lavasession.RPCStaticProviderEndpoint)
			for _, provider := range tt.providers {
				providerMap[provider.Name] = provider
			}

			for i, session := range sessionSlice {
				require.Equal(t, expectedNamesSorted[i], session.PublicLavaAddress)
				require.True(t, session.StaticProvider, "Should be marked as static provider")
				require.Equal(t, uint64(math.MaxUint64/2), session.MaxComputeUnits)
				require.Equal(t, tt.epoch, session.PairingEpoch)

				// Find the matching provider config by name
				provider := providerMap[session.PublicLavaAddress]
				require.NotNil(t, provider, "Provider config should exist for %s", session.PublicLavaAddress)

				// Verify endpoints were created correctly
				endpoints := session.Endpoints
				require.Len(t, endpoints, len(provider.NodeUrls))

				// Verify each endpoint has the correct configuration
				for j, endpoint := range endpoints {
					require.Equal(t, provider.NodeUrls[j].Url, endpoint.NetworkAddress)
				}
			}
		})
	}
}

func TestCreateProviderSessionsFromConfig_EmptyProviders(t *testing.T) {
	pu := &PairingUpdater{}
	providers := []*lavasession.RPCStaticProviderEndpoint{}

	rpcEndpoint := lavasession.RPCEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
	}
	existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	sessions := pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 100, existingProviders, "StaticProvider_")

	require.Empty(t, sessions)
}

func TestCreateProviderSessionsFromConfig_ProviderNames(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		expected     string
	}{
		{
			name:         "custom name",
			providerName: "CustomProviderName",
			expected:     "CustomProviderName",
		},
		{
			name:         "name with special characters",
			providerName: "Provider-1_Test.2024",
			expected:     "Provider-1_Test.2024",
		},
		{
			name:         "name with unicode",
			providerName: "Provider-测试-Тест",
			expected:     "Provider-测试-Тест",
		},
		{
			name:         "very long name",
			providerName: "VeryLongProviderNameThatContainsLotsOfCharactersAndShouldStillWork",
			expected:     "VeryLongProviderNameThatContainsLotsOfCharactersAndShouldStillWork",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pu := &PairingUpdater{}
			providers := []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls: []common.NodeUrl{
						{Url: "https://example.com"},
					},
					Geolocation: 1,
					Name:        tt.providerName,
				},
			}

			rpcEndpoint := lavasession.RPCEndpoint{
				ChainID:      "ETH1",
				ApiInterface: "jsonrpc",
			}
			existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

			sessions := pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 100, existingProviders, "StaticProvider_")

			require.Len(t, sessions, 1)
			for _, session := range sessions {
				require.Equal(t, tt.expected, session.PublicLavaAddress)
				require.True(t, session.StaticProvider, "Should be marked as static provider")
			}
		})
	}
}

func TestCreateProviderSessionsFromConfig_NetworkAddressFormatting(t *testing.T) {
	tests := []struct {
		name        string
		nodeUrls    []common.NodeUrl
		expectedLen int
	}{
		{
			name: "single HTTP URL",
			nodeUrls: []common.NodeUrl{
				{Url: "https://example.com"},
			},
			expectedLen: 1,
		},
		{
			name: "multiple URLs with different protocols",
			nodeUrls: []common.NodeUrl{
				{Url: "https://http.example.com"},
				{Url: "wss://ws.example.com"},
				{Url: "http://plain.example.com"},
			},
			expectedLen: 3,
		},
		{
			name: "URLs with ports",
			nodeUrls: []common.NodeUrl{
				{Url: "https://example.com:8545"},
				{Url: "wss://example.com:8546"},
			},
			expectedLen: 2,
		},
		{
			name: "URLs with paths",
			nodeUrls: []common.NodeUrl{
				{Url: "https://example.com/v1/rpc"},
				{Url: "https://example.com/api/v2"},
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pu := &PairingUpdater{}
			providers := []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls:     tt.nodeUrls,
					Geolocation:  1,
					Name:         "TestProvider",
				},
			}

			rpcEndpoint := lavasession.RPCEndpoint{
				ChainID:      "ETH1",
				ApiInterface: "jsonrpc",
			}
			existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

			sessions := pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 100, existingProviders, "StaticProvider_")

			require.Len(t, sessions, 1)
			for _, session := range sessions {
				require.Len(t, session.Endpoints, tt.expectedLen)

				// Verify each endpoint has the correct network address
				for i, endpoint := range session.Endpoints {
					require.Equal(t, tt.nodeUrls[i].Url, endpoint.NetworkAddress)
				}
			}
		})
	}
}

// Test to ensure static provider sessions have the correct properties for availability
func TestCreateProviderSessionsFromConfig_SessionProperties(t *testing.T) {
	pu := &PairingUpdater{}
	providers := []*lavasession.RPCStaticProviderEndpoint{
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls: []common.NodeUrl{
				{Url: "https://example.com"},
			},
			Geolocation: 1,
			Name:        "TestProvider",
		},
	}

	rpcEndpoint := lavasession.RPCEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
	}
	existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	sessions := pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 500, existingProviders, "StaticProvider_")

	require.Len(t, sessions, 1)
	for _, session := range sessions {
		// Verify high availability properties
		require.Equal(t, uint64(math.MaxUint64/2), session.MaxComputeUnits, "Should have high compute units for availability")
		require.Equal(t, uint64(500), session.PairingEpoch, "Should use correct epoch")
		require.Equal(t, "TestProvider", session.PublicLavaAddress, "Should use correct provider name")
		require.True(t, session.StaticProvider, "Should be marked as static provider")

		// Verify endpoint properties
		require.Len(t, session.Endpoints, 1)
		endpoint := session.Endpoints[0]
		require.Equal(t, "https://example.com", endpoint.NetworkAddress)
		require.Empty(t, endpoint.Connections, "Connections should be empty initially")
	}
}

// Benchmark test for provider session creation
func BenchmarkCreateProviderSessionsFromConfig(b *testing.B) {
	pu := &PairingUpdater{}
	providers := []*lavasession.RPCStaticProviderEndpoint{
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls: []common.NodeUrl{
				{Url: "https://example1.com"},
				{Url: "https://example2.com"},
			},
			Geolocation: 1,
			Name:        "BenchProvider1",
		},
		{
			ChainID:      "ETH1", // Changed to ETH1 to match rpcEndpoint
			ApiInterface: "jsonrpc",
			NodeUrls: []common.NodeUrl{
				{Url: "https://polygon.example.com"},
			},
			Geolocation: 1,
			Name:        "BenchProvider2",
		},
	}

	rpcEndpoint := lavasession.RPCEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
	}
	existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 100, existingProviders, "StaticProvider_")
	}
}
