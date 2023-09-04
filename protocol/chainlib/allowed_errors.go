package chainlib

// AllowedErrorsMap is a map of allowed errors for various API interfaces
var AllowedErrorsMap = map[string]map[string]string{
	"jsonrpc": {
		"-32700": "Parse error",
		"-32600": "Invalid request",
		"-32602": "Invalid params",
		"-32603": "Internal error",
		"-32000": "Invalid input",
		"-32001": "Resource not found",
		"-32002": "Resource unavailable",
		"-32003": "Transaction rejected",
		"-32006": "JSON-RPC version not supported",
	},
	"rest": {
		// Add REST-specific allowed errors here
	},
	"grpc": {
		// Add gRPC-specific allowed errors here
	},
	"tendermintrpc": {
		// Add tendermint-rpc-specific allowed errors here
	},
}
