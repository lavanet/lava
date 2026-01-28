package relaycore

type Selection int

const (
	MaxCallsPerRelay = 50
)

var RelayCountOnNodeError = 2
var RelayCountOnProtocolError = 2

// selection Enum, do not add other const
const (
	Quorum     Selection = iota // get the majority out of requiredSuccesses
	BestResult                  // get the best result, even if it means waiting
)
