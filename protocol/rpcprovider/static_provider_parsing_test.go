package rpcprovider

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestParseStaticProviderEndpoints_ValidConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		geolocation    uint64
		expectedCount  int
		expectedNames  []string
		expectedChains []string
	}{
		{
			name: "single valid provider",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"name":          "EthProvider1",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth1.example.com"},
						},
					},
				},
			},
			geolocation:    1,
			expectedCount:  1,
			expectedNames:  []string{"EthProvider1"},
			expectedChains: []string{"ETH1"},
		},
		{
			name: "multiple valid providers",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"name":          "Primary-Ethereum",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth-primary.example.com"},
						},
					},
					{
						"name":          "Secondary-Polygon",
						"chain-id":      "POLYGON1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://polygon.example.com"},
						},
					},
				},
			},
			geolocation:    2,
			expectedCount:  2,
			expectedNames:  []string{"Primary-Ethereum", "Secondary-Polygon"},
			expectedChains: []string{"ETH1", "POLYGON1"},
		},
		{
			name: "provider with multiple node urls",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"name":          "MultiNode-Provider",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://node1.example.com"},
							{"url": "https://node2.example.com"},
							{"url": "wss://ws.example.com"},
						},
					},
				},
			},
			geolocation:    1,
			expectedCount:  1,
			expectedNames:  []string{"MultiNode-Provider"},
			expectedChains: []string{"ETH1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			for key, value := range tt.config {
				v.Set(key, value)
			}

			endpoints, err := ParseStaticProviderEndpoints(v, "static-providers", tt.geolocation)

			require.NoError(t, err)
			require.Len(t, endpoints, tt.expectedCount)

			for i, endpoint := range endpoints {
				require.Equal(t, tt.expectedNames[i], endpoint.Name)
				require.Equal(t, tt.expectedChains[i], endpoint.ChainID)
				require.Equal(t, tt.geolocation, endpoint.Geolocation)
				require.NoError(t, endpoint.Validate())
			}
		})
	}
}

func TestParseStaticProviderEndpoints_InvalidConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		geolocation   uint64
		expectError   bool
		errorContains string
	}{
		{
			name: "missing provider name",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth1.example.com"},
						},
					},
				},
			},
			geolocation:   1,
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
		{
			name: "empty provider name",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"name":          "",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth1.example.com"},
						},
					},
				},
			},
			geolocation:   1,
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
		{
			name: "mixed valid and invalid providers",
			config: map[string]interface{}{
				"static-providers": []map[string]interface{}{
					{
						"name":          "ValidProvider",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://valid.example.com"},
						},
					},
					{
						"chain-id":      "ETH1", // Missing name
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://invalid.example.com"},
						},
					},
				},
			},
			geolocation:   1,
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			for key, value := range tt.config {
				v.Set(key, value)
			}

			endpoints, err := ParseStaticProviderEndpoints(v, "static-providers", tt.geolocation)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				require.Nil(t, endpoints)
			} else {
				require.NoError(t, err)
				require.NotNil(t, endpoints)
			}
		})
	}
}

func TestParseStaticProviderEndpoints_EmptyConfig(t *testing.T) {
	v := viper.New()
	v.Set("static-providers", []interface{}{})

	endpoints, err := ParseStaticProviderEndpoints(v, "static-providers", 1)

	require.NoError(t, err)
	require.Empty(t, endpoints)
}

func TestParseStaticProviderEndpoints_GeolocationHandling(t *testing.T) {
	config := map[string]interface{}{
		"static-providers": []map[string]interface{}{
			{
				"name":          "GeoTestProvider",
				"chain-id":      "ETH1",
				"api-interface": "jsonrpc",
				"node-urls": []map[string]interface{}{
					{"url": "https://example.com"},
				},
			},
		},
	}

	tests := []struct {
		name        string
		geolocation uint64
	}{
		{"geolocation 0", 0},
		{"geolocation 1", 1},
		{"geolocation 100", 100},
		{"max geolocation", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			for key, value := range config {
				v.Set(key, value)
			}

			endpoints, err := ParseStaticProviderEndpoints(v, "static-providers", tt.geolocation)

			require.NoError(t, err)
			require.Len(t, endpoints, 1)
			require.Equal(t, tt.geolocation, endpoints[0].Geolocation)
		})
	}
}

func TestParseStaticProviderEndpoints_ComplexConfig(t *testing.T) {
	config := map[string]interface{}{
		"static-providers": []map[string]interface{}{
			{
				"name":          "Complex-Provider_1",
				"chain-id":      "ETH1",
				"api-interface": "jsonrpc",
				"node-urls": []map[string]interface{}{
					{
						"url": "https://primary.example.com",
						"auth-config": map[string]interface{}{
							"auth-type": "bearer",
							"auth-key":  "secret-key",
						},
						"addons": []string{"debug", "trace"},
					},
					{
						"url": "wss://websocket.example.com",
					},
				},
			},
			{
				"name":          "Provider-With-Special-Chars_测试",
				"chain-id":      "POLYGON1",
				"api-interface": "rest",
				"node-urls": []map[string]interface{}{
					{"url": "https://polygon-rest.example.com/v1"},
				},
			},
		},
	}

	v := viper.New()
	for key, value := range config {
		v.Set(key, value)
	}

	endpoints, err := ParseStaticProviderEndpoints(v, "static-providers", 1)

	require.NoError(t, err)
	require.Len(t, endpoints, 2)

	// Verify first provider
	require.Equal(t, "Complex-Provider_1", endpoints[0].Name)
	require.Equal(t, "ETH1", endpoints[0].ChainID)
	require.Equal(t, "jsonrpc", endpoints[0].ApiInterface)
	require.Len(t, endpoints[0].NodeUrls, 2)

	// Verify second provider
	require.Equal(t, "Provider-With-Special-Chars_测试", endpoints[1].Name)
	require.Equal(t, "POLYGON1", endpoints[1].ChainID)
	require.Equal(t, "rest", endpoints[1].ApiInterface)
	require.Len(t, endpoints[1].NodeUrls, 1)

	// Verify both providers pass validation
	for _, endpoint := range endpoints {
		require.NoError(t, endpoint.Validate())
	}
}

// Benchmark test for parsing performance
func BenchmarkParseStaticProviderEndpoints(b *testing.B) {
	config := map[string]interface{}{
		"static-providers": []map[string]interface{}{
			{
				"name":          "BenchProvider1",
				"chain-id":      "ETH1",
				"api-interface": "jsonrpc",
				"node-urls": []map[string]interface{}{
					{"url": "https://example1.com"},
				},
			},
			{
				"name":          "BenchProvider2",
				"chain-id":      "POLYGON1",
				"api-interface": "jsonrpc",
				"node-urls": []map[string]interface{}{
					{"url": "https://example2.com"},
				},
			},
		},
	}

	v := viper.New()
	for key, value := range config {
		v.Set(key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseStaticProviderEndpoints(v, "static-providers", 1)
	}
}
