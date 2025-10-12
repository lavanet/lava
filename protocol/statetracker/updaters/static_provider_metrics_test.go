package updaters

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/stretchr/testify/require"
)

// TestStaticProviderUsesNameInMetrics verifies that static providers have their name
// (not address) set in PublicLavaAddress, which is used for consumer metrics
func TestStaticProviderUsesNameInMetrics(t *testing.T) {
	tests := []struct {
		name              string
		providerName      string
		expectNameInField bool
		description       string
	}{
		{
			name:              "static provider with name",
			providerName:      "MyEthereumProvider",
			expectNameInField: true,
			description:       "Static provider should use name as PublicLavaAddress for metrics",
		},
		{
			name:              "static provider with descriptive name",
			providerName:      "Production-Ethereum-Node-1",
			expectNameInField: true,
			description:       "Static provider with descriptive name should be used in metrics",
		},
		{
			name:              "static provider with special characters",
			providerName:      "Provider_2024-Q1.Primary",
			expectNameInField: true,
			description:       "Special characters in name should be preserved",
		},
		{
			name:              "static provider with unicode",
			providerName:      "测试提供者-Provider",
			expectNameInField: true,
			description:       "Unicode characters in name should be preserved",
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
						{Url: "https://eth.example.com"},
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

			require.Len(t, sessions, 1, "Should create exactly one session")

			// Verify the session
			for _, session := range sessions {
				// This is the key assertion: PublicLavaAddress should contain the provider name
				require.Equal(t, tt.providerName, session.PublicLavaAddress, tt.description)

				// Verify it's marked as static
				require.True(t, session.StaticProvider, "Should be marked as static provider")

				// The PublicLavaAddress should NOT be a blockchain address format
				// (blockchain addresses start with "lava1" or similar prefixes)
				require.NotContains(t, session.PublicLavaAddress, "lava1", "Should not be a blockchain address")
			}
		})
	}
}

// TestStaticProviderPublicAddressUsedInMetricsFlow simulates how consumer metrics
// will access the provider identifier through PublicLavaAddress
func TestStaticProviderPublicAddressUsedInMetricsFlow(t *testing.T) {
	pu := &PairingUpdater{}

	// Create static providers with different names
	providers := []*lavasession.RPCStaticProviderEndpoint{
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls:     []common.NodeUrl{{Url: "https://eth1.example.com"}},
			Geolocation:  1,
			Name:         "Ethereum-Provider-Primary",
		},
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls:     []common.NodeUrl{{Url: "https://eth2.example.com"}},
			Geolocation:  1,
			Name:         "Ethereum-Provider-Secondary",
		},
	}

	rpcEndpoint := lavasession.RPCEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
	}
	existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	sessions := pu.createProviderSessionsFromConfig(providers, rpcEndpoint, 100, existingProviders, "StaticProvider_")

	require.Len(t, sessions, 2, "Should create two sessions")

	// Simulate what consumer metrics code does: iterate sessions and use PublicLavaAddress
	providerIdentifiers := []string{}
	for _, session := range sessions {
		// This simulates the exact pattern used in consumer code:
		// providerPublicAddress := session.PublicLavaAddress
		providerIdentifier := session.PublicLavaAddress
		providerIdentifiers = append(providerIdentifiers, providerIdentifier)
	}

	// Verify we got the provider names, not addresses
	require.Contains(t, providerIdentifiers, "Ethereum-Provider-Primary", "Should contain first provider name")
	require.Contains(t, providerIdentifiers, "Ethereum-Provider-Secondary", "Should contain second provider name")

	// Verify these are human-readable names, not addresses
	for _, identifier := range providerIdentifiers {
		require.NotEmpty(t, identifier, "Identifier should not be empty")
		require.NotContains(t, identifier, "lava1", "Should not be blockchain address format")
		require.Contains(t, identifier, "Ethereum-Provider", "Should contain descriptive name")
	}
}

// TestStaticProviderVsRegularProviderInMetrics verifies the difference between
// how static providers and regular providers appear in metrics
func TestStaticProviderVsRegularProviderInMetrics(t *testing.T) {
	pu := &PairingUpdater{}

	// Create a static provider
	staticProviders := []*lavasession.RPCStaticProviderEndpoint{
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls:     []common.NodeUrl{{Url: "https://static.example.com"}},
			Geolocation:  1,
			Name:         "Static-Provider-Name",
		},
	}

	rpcEndpoint := lavasession.RPCEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
	}
	existingProviders := map[uint64]*lavasession.ConsumerSessionsWithProvider{}

	staticSessions := pu.createProviderSessionsFromConfig(staticProviders, rpcEndpoint, 100, existingProviders, "StaticProvider_")

	require.Len(t, staticSessions, 1)

	// Get the static provider identifier
	var staticIdentifier string
	for _, session := range staticSessions {
		staticIdentifier = session.PublicLavaAddress
		require.True(t, session.StaticProvider, "Should be marked as static")
	}

	// Verify static provider uses name
	require.Equal(t, "Static-Provider-Name", staticIdentifier, "Static provider should use configured name")

	// Note: Regular providers would use NewConsumerSessionWithProvider(address, ...)
	// where address is a blockchain address like "lava1..."
	// Static providers use NewConsumerSessionWithProvider(name, ...) where name is the configured name
	require.NotContains(t, staticIdentifier, "lava1", "Static provider should not use blockchain address format")
}

// TestConsumerMetricsKeyGeneration verifies how metrics keys are generated for static providers
func TestConsumerMetricsKeyGeneration(t *testing.T) {
	pu := &PairingUpdater{}

	providers := []*lavasession.RPCStaticProviderEndpoint{
		{
			ChainID:      "ETH1",
			ApiInterface: "jsonrpc",
			NodeUrls:     []common.NodeUrl{{Url: "https://example.com"}},
			Geolocation:  1,
			Name:         "MyProvider",
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
		// Simulate how consumer code generates metrics keys
		// Example from consumer_metrics_manager.go line 509:
		// providerRelaysKey := providerAddress + apiInterface
		providerAddress := session.PublicLavaAddress
		apiInterface := "jsonrpc"
		metricsKey := providerAddress + apiInterface

		// For static provider, this key should contain the name
		require.Equal(t, "MyProviderjsonrpc", metricsKey, "Metrics key should use provider name")
		require.Contains(t, metricsKey, "MyProvider", "Metrics key should contain provider name")

		// Verify provider endpoint for liveness metrics
		providerEndpoint := session.Endpoints[0].NetworkAddress
		require.Equal(t, "https://example.com", providerEndpoint, "Should have correct endpoint")
	}
}

// TestStaticProviderNamePreservationThroughFlow ensures the provider name
// is preserved from config through to the session object used by metrics
func TestStaticProviderNamePreservationThroughFlow(t *testing.T) {
	testCases := []struct {
		name         string
		providerName string
	}{
		{"simple name", "SimpleProvider"},
		{"name with dashes", "Provider-With-Dashes"},
		{"name with underscores", "Provider_With_Underscores"},
		{"name with dots", "Provider.With.Dots"},
		{"name with numbers", "Provider123"},
		{"mixed format", "Provider-2024_Q1.Primary"},
		{"long descriptive name", "Production-Ethereum-Mainnet-Provider-Primary-Instance"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pu := &PairingUpdater{}

			providers := []*lavasession.RPCStaticProviderEndpoint{
				{
					ChainID:      "ETH1",
					ApiInterface: "jsonrpc",
					NodeUrls:     []common.NodeUrl{{Url: "https://example.com"}},
					Geolocation:  1,
					Name:         tc.providerName,
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
				// The name should be exactly preserved
				require.Equal(t, tc.providerName, session.PublicLavaAddress,
					"Provider name should be exactly preserved in PublicLavaAddress")

				// Verify it's accessible as the "provider address" for metrics
				providerAddressForMetrics := session.PublicLavaAddress
				require.Equal(t, tc.providerName, providerAddressForMetrics,
					"Name should be accessible as provider address for metrics")
			}
		})
	}
}
