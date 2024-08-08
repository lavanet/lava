package format

import (
	"testing"

	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestFormatter(t *testing.T) {
	inputFormatter, outputFormatter := FormatterForRelayRequestAndResponse(spectypes.APIInterfaceJsonRPC)
	data := `[{"jsonrpc":"2.0","id":1,"method":"eth_chainId"},{"jsonrpc":"2.0","id":3,"method":"eth_chainId"}]`
	inputData := inputFormatter([]byte(data))
	require.NotEqual(t, string(inputData), data)
	outData := outputFormatter(inputData)
	require.Equal(t, string(outData), data)
}
