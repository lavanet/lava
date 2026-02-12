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

// Selection Enum - defines relay behavior modes
const (
	Stateless       Selection = iota // Single provider with retries on failure, sequential provider attempts until success
	Stateful                         // All top providers at once, waits for best result (no retries) or for all providers to return a response
	CrossValidation                  // maxParticipants providers at once, no retries, waits for agreementThreshold matching responses
)
