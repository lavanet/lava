package qos

import sdk "github.com/cosmos/cosmos-sdk/types"

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(1, 1) // TODO move to params pairing
const (
	PercentileToCalculateLatency = 0.9
	MinProvidersForSync          = 0.6
)

type DegradeAvailabilityReputation interface{}

type SendQoSUpdate interface{}
