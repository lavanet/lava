package lavasession

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxConsecutiveConnectionAttempts                 = 3
	TimeoutForEstablishingAConnectionInMS            = 300 * time.Millisecond
	MaxSessionsAllowedPerProvider                    = 10 // Max number of sessions allowed per provider
	MaxAllowedBlockListedSessionPerProvider          = 3
	MaximumNumberOfFailuresAllowedPerConsumerSession = 2
)

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(5, 2) //TODO move to params pairing
const (
	PercentileToCalculateLatency = 0.9
	MinProvidersForSync          = 0.6
	LatencyThresholdStatic       = 1 * time.Second
	LatencyThresholdSlope        = 1 * time.Millisecond
	StaleEpochDistance           = 3 // relays done 3 epochs back are ready to be rewarded

)
