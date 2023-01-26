package lavasession

import (
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MaxConsecutiveConnectionAttempts                 = 10
	TimeoutForEstablishingAConnection                = 1 * time.Second
	MaxSessionsAllowedPerProvider                    = 1000 // Max number of sessions allowed per provider
	MaxAllowedBlockListedSessionPerProvider          = 3
	MaximumNumberOfFailuresAllowedPerConsumerSession = 3
	RelayNumberIncrement                             = 1
	DataReliabilitySessionId                         = 0 // data reliability session id is 0. we can change to more sessions later if needed.
	DataReliabilityCuSum                             = 0
	GeolocationFlag                                  = "geolocation"
)

var AvailabilityPercentage sdk.Dec = sdk.NewDecWithPrec(5, 2) // TODO move to params pairing
const (
	PercentileToCalculateLatency = 0.9
	MinProvidersForSync          = 0.6
	LatencyThresholdStatic       = 1 * time.Second
	LatencyThresholdSlope        = 1 * time.Millisecond
	StaleEpochDistance           = 3 // relays done 3 epochs back are ready to be rewarded

)

func PrintRPCEndpoint(endpoint *RPCEndpoint) (retStr string) {
	retStr = endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10)
	return
}
func PrintRPCProviderEndpoint(endpoint *RPCProviderEndpoint) (retStr string) {
	retStr = endpoint.ChainID + ":" + endpoint.ApiInterface + " Network Address:" + endpoint.NetworkAddress + "Node: " + endpoint.NodeUrl + " Geolocation:" + strconv.FormatUint(endpoint.Geolocation, 10)
	return
}
