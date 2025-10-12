package rpcprovider

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestParseStaticProviderEndpoints_RequiresName(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid provider with name",
			config: map[string]interface{}{
				"endpoints": []map[string]interface{}{
					{
						"name":          "MyProvider",
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth1.example.com"},
						},
					},
				},
			},
			shouldError: false,
		},
		{
			name: "missing provider name should fail",
			config: map[string]interface{}{
				"endpoints": []map[string]interface{}{
					{
						"chain-id":      "ETH1",
						"api-interface": "jsonrpc",
						"node-urls": []map[string]interface{}{
							{"url": "https://eth1.example.com"},
						},
					},
				},
			},
			shouldError: true,
			errorMsg:    "provider name cannot be empty",
		},
		{
			name: "empty provider name should fail",
			config: map[string]interface{}{
				"endpoints": []map[string]interface{}{
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
			shouldError: true,
			errorMsg:    "provider name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			for key, value := range tt.config {
				v.Set(key, value)
			}

			endpoints, err := ParseStaticProviderEndpoints(v, "endpoints", 1)

			if tt.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, endpoints)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, endpoints)
				require.NotEmpty(t, endpoints[0].Name)
			}
		})
	}
}
