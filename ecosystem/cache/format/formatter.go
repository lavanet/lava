package format

import (
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

// input formatter works on the input data
func FormatterForRelayRequestAndResponse(apiInterface string) (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	switch apiInterface {
	case spectypes.APIInterfaceJsonRPC:
		return FormatterForRelayRequestAndResponseJsonRPC()
	case spectypes.APIInterfaceTendermintRPC:
		// tendermint has json rpc input as well
		return FormatterForRelayRequestAndResponseJsonRPC()
	default:
		return IdentityFormatter()
	}
}

func IdentityFormatter() (inputFormatter func([]byte) []byte, outputFormatter func([]byte) []byte) {
	inputFormatter = func(inpData []byte) []byte {
		return inpData
	}
	outputFormatter = inputFormatter
	return
}
