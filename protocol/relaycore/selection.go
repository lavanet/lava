package relaycore

type Selection int

const (
	MaxCallsPerRelay = 50
)

var RelayCountOnNodeError = 2

// selection Enum, do not add other const
const (
	Stateless Selection = iota // retries enabled, seeks majority consensus from node responses
	Stateful                   // all top providers at once, waits for best result (no retries)
)
