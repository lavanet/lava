package format

import (
	spectypes "github.com/lavanet/lava/v5/types/spec"
)

// FormatterForRelayRequestAndResponse returns input/output formatter functions
// for the given API interface, used to normalize requests for cache key hashing.
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
	return inputFormatter, outputFormatter
}
