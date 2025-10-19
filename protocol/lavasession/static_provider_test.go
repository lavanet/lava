package lavasession

import (
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestRPCStaticProviderEndpoint_Validate(t *testing.T) {
	tests := []struct {
		name          string
		providerName  string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid provider name",
			providerName: "ValidProvider",
			expectError:  false,
		},
		{
			name:         "valid provider name with special chars",
			providerName: "Provider-1_Test",
			expectError:  false,
		},
		{
			name:         "valid provider name with spaces",
			providerName: "Provider With Spaces",
			expectError:  false,
		},
		{
			name:          "empty provider name",
			providerName:  "",
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
		{
			name:         "whitespace only provider name",
			providerName: "   ",
			expectError:  false, // Note: this currently passes validation, may want to trim whitespace
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &RPCStaticProviderEndpoint{
				ChainID:      "ETH1",
				ApiInterface: "jsonrpc",
				NodeUrls: []common.NodeUrl{
					{Url: "https://example.com"},
				},
				Name: tt.providerName,
			}

			err := endpoint.Validate()

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRPCStaticProviderEndpoint_ToBase(t *testing.T) {
	staticEndpoint := &RPCStaticProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NodeUrls: []common.NodeUrl{
			{Url: "https://example.com"},
		},
		Geolocation: 1,
		Name:        "TestProvider",
	}

	baseEndpoint := staticEndpoint.ToBase()

	// Verify that the base endpoint has all the original fields
	require.Equal(t, staticEndpoint.ChainID, baseEndpoint.ChainID)
	require.Equal(t, staticEndpoint.ApiInterface, baseEndpoint.ApiInterface)
	require.Equal(t, staticEndpoint.NodeUrls, baseEndpoint.NodeUrls)
	require.Equal(t, staticEndpoint.Geolocation, baseEndpoint.Geolocation)

	// Verify that the base endpoint is a new instance with copied values
	require.NotEqual(t, staticEndpoint, baseEndpoint)
}

func TestRPCStaticProviderEndpoint_Name_Access(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		expected     string
	}{
		{
			name:         "simple name",
			providerName: "SimpleProvider",
			expected:     "SimpleProvider",
		},
		{
			name:         "name with special characters",
			providerName: "Provider-1_Test.2024",
			expected:     "Provider-1_Test.2024",
		},
		{
			name:         "name with unicode characters",
			providerName: "Provider-测试-Тест",
			expected:     "Provider-测试-Тест",
		},
		{
			name:         "empty name",
			providerName: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := &RPCStaticProviderEndpoint{
				Name: tt.providerName,
			}

			require.Equal(t, tt.expected, endpoint.Name)
		})
	}
}

func TestRPCStaticProviderEndpoint_CompleteConfig(t *testing.T) {
	// Test a complete valid configuration
	endpoint := &RPCStaticProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NodeUrls: []common.NodeUrl{
			{Url: "https://primary.example.com"},
			{Url: "https://backup.example.com"},
		},
		Geolocation: 1,
		Name:        "EthereumMainnetProvider",
	}

	// Should pass validation
	require.NoError(t, endpoint.Validate())

	// Should have correct name
	require.Equal(t, "EthereumMainnetProvider", endpoint.Name)

	// Should be able to convert to base
	base := endpoint.ToBase()
	require.NotNil(t, base)
	require.Equal(t, "ETH1", base.ChainID)
	require.Equal(t, "jsonrpc", base.ApiInterface)
	require.Len(t, base.NodeUrls, 2)
}

// Benchmark tests for validation performance
func BenchmarkRPCStaticProviderEndpoint_Validate(b *testing.B) {
	endpoint := &RPCStaticProviderEndpoint{
		ChainID:      "ETH1",
		ApiInterface: "jsonrpc",
		NodeUrls: []common.NodeUrl{
			{Url: "https://example.com"},
		},
		Name: "BenchmarkProvider",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = endpoint.Validate()
	}
}

func BenchmarkRPCStaticProviderEndpoint_ValidateEmpty(b *testing.B) {
	endpoint := &RPCStaticProviderEndpoint{
		Name: "", // Empty name to trigger error path
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = endpoint.Validate()
	}
}
