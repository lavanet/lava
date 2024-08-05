package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
)

func (qos *QualityOfServiceReport) ComputeQoS() (sdk.Dec, error) {
	if qos.Availability.GT(sdk.OneDec()) || qos.Availability.LT(sdk.ZeroDec()) ||
		qos.Latency.GT(sdk.OneDec()) || qos.Latency.LT(sdk.ZeroDec()) ||
		qos.Sync.GT(sdk.OneDec()) || qos.Sync.LT(sdk.ZeroDec()) {
		return sdk.ZeroDec(), fmt.Errorf("QoS scores is not between 0-1")
	}

	return qos.Availability.Mul(qos.Sync).Mul(qos.Latency).ApproxRoot(3)
}

func (qos *QualityOfServiceReport) ComputeQoSExcellence() (sdk.Dec, error) {
	if qos.Availability.LTE(sdk.ZeroDec()) ||
		qos.Latency.LTE(sdk.ZeroDec()) ||
		qos.Sync.LTE(sdk.ZeroDec()) {
		return sdk.ZeroDec(), fmt.Errorf("QoS excellence scores is below 0")
	}
	return qos.Availability.Quo(qos.Sync).Quo(qos.Latency).ApproxRoot(3)
}

// ComputeQosExcellenceForReputation computes the score from the QoS excellence report to update the provider's reputation
// report score = latency + sync*syncFactor + ((1/availability) - 1) * FailureCost (note: larger value is worse)
func (qos QualityOfServiceReport) ComputeQosExcellenceForReputation(syncFactor math.LegacyDec) (math.LegacyDec, error) {
	if qos.Availability.LT(sdk.ZeroDec()) ||
		qos.Latency.LT(sdk.ZeroDec()) ||
		qos.Sync.LT(sdk.ZeroDec()) || syncFactor.LT(sdk.ZeroDec()) {
		return sdk.ZeroDec(), utils.LavaFormatWarning("ComputeQosExcellenceForReputation: compute failed", fmt.Errorf("QoS excellence scores is below 0"),
			utils.LogAttr("availability", qos.Availability.String()),
			utils.LogAttr("sync", qos.Sync.String()),
			utils.LogAttr("latency", qos.Latency.String()),
		)
	}

	latency := qos.Latency
	sync := qos.Sync.Mul(syncFactor)
	availability := math.LegacyNewDec(FailureCost)
	if !qos.Availability.IsZero() {
		availability = availability.Mul((math.LegacyOneDec().Quo(qos.Availability)).Sub(math.LegacyOneDec()))
	} else {
		availability = math.LegacyMaxSortableDec.QuoInt64(2) // on qs.Availability = 0 we take the largest score possible
	}
	return latency.Add(sync).Add(availability), nil
}
