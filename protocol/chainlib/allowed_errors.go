package chainlib

// AllowedErrorsMap is a map of allowed errors for various API interfaces
var AllowedErrorsMap = map[string]map[string]string{
	// json-rpc-specific allowed errors
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
		// REST-specific allowed errors
		"2":  "tx parse error",                        // client's fault for sending malformed transaction
		"3":  "invalid sequence",                      // due to incorrect nonce from client
		"6":  "unknown request",                       // client is asking for something unknown
		"7":  "invalid address",                       // client provided an invalid address
		"8":  "invalid pubkey",                        // client provided an invalid public key
		"9":  "unknown address",                       // client provided an address that is not known
		"10": "invalid coins",                         // client provided invalid coin/token information
		"11": "out of gas",                            // client did not supply enough gas
		"12": "memo too large",                        // client included a memo that is too large
		"13": "insufficient fee",                      // client did not provide sufficient fees
		"14": "maximum number of signatures exceeded", // client provided too many signatures
		"15": "no signatures supplied",                // client did not provide any signatures
		"18": "invalid request",                       // client sent a request that couldn't be understood
		"21": "tx too large",                          // client sent a too-large transaction
		"26": "invalid height",                        // client specified an invalid block height
		"27": "invalid version",                       // client is using an unsupported version
		"28": "invalid chain-id",                      // client provided invalid chain-id
		"29": "invalid type",                          // client used an invalid type in request
		"30": "tx timeout height",                     // client set an invalid timeout height for the transaction
		"32": "incorrect account sequence",            // client used an incorrect sequence number for the account
		"37": "feature not supported",                 // client tried to use a feature that is not supported
		"38": "not found",                             // client is asking for a resource that doesn't exist
		"41": "invalid gas limit",                     // client set an invalid gas limit
	},
	"tendermintrpc": {
		// tendermint-rpc-specific allowed errors
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
	"grpc": {
		// gRPC-specific allowed errors
	},
}
