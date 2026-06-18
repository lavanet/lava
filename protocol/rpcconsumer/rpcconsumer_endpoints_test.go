package rpcconsumer

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Test 2: the per-chain max-provider-latency field is parsed from the YAML endpoints
// config into RPCEndpoint.MaxProviderLatency, and defaults to 0 when omitted.
func TestParseEndpoints_MaxProviderLatency(t *testing.T) {
	yamlConfig := `
endpoints:
  - chain-id: ETH1
    api-interface: jsonrpc
    network-address: 127.0.0.1:3333
    max-provider-latency: 1.5
  - chain-id: LAV1
    api-interface: rest
    network-address: 127.0.0.1:3340
`
	v := viper.New()
	v.SetConfigType("yml")
	require.NoError(t, v.ReadConfig(strings.NewReader(yamlConfig)))

	endpoints, err := ParseEndpoints(v, 1)
	require.NoError(t, err)
	require.Len(t, endpoints, 2)

	require.Equal(t, "ETH1", endpoints[0].ChainID)
	require.Equal(t, 1.5, endpoints[0].MaxProviderLatency, "max-provider-latency must be parsed from YAML")

	require.Equal(t, "LAV1", endpoints[1].ChainID)
	require.Equal(t, float64(0), endpoints[1].MaxProviderLatency, "omitted max-provider-latency must default to 0 (disabled)")
}
