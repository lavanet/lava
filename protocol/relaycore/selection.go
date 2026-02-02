package relaycore

type Selection int

const (
	MaxCallsPerRelay = 50
)

var RelayCountOnNodeError = 2

// DisableBatchRequestRetry prevents batch requests from being retried when set to true.
// Batch requests (JSON-RPC batches) cannot be hashed for caching, so retries may be unnecessary.
// This is controlled via the --disable-batch-request-retry flag.
// Disabled by default (true) because batch retries can cause issues with stateful operations.
var DisableBatchRequestRetry = true

// selection Enum, do not add other const
const (
	Stateless Selection = iota // Retries enabled, sequential provider attempts, seeks majority consensus
	Stateful                   // all top providers at once, waits for best result (no retries) or for all the providers to return a response
)
