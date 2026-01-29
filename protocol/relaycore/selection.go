package relaycore

type Selection int

const (
	MaxCallsPerRelay = 50
)

var RelayCountOnNodeError = 2

// selection Enum, do not add other const
const (
	Stateless Selection = iota // Retries enabled, sequential provider attempts, seeks majority consensus
	Stateful                   // all top providers at once, waits for best result (no retries) or for all the providers to return a response
)
