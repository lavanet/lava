package types

import (
	"fmt"

	"cosmossdk.io/math"
)

func (qos *QualityOfServiceReport) ComputeQoS() (math.LegacyDec, error) {
	if qos.Availability.GT(math.LegacyOneDec()) || qos.Availability.LT(math.LegacyZeroDec()) ||
		qos.Latency.GT(math.LegacyOneDec()) || qos.Latency.LT(math.LegacyZeroDec()) ||
		qos.Sync.GT(math.LegacyOneDec()) || qos.Sync.LT(math.LegacyZeroDec()) {
		return math.LegacyZeroDec(), fmt.Errorf("QoS scores is not between 0-1")
	}

	return qos.Availability.Mul(qos.Sync).Mul(qos.Latency).ApproxRoot(3)
}

func (qos *QualityOfServiceReport) ComputeQoSExcellence() (math.LegacyDec, error) {
	if qos.Availability.LTE(math.LegacyZeroDec()) ||
		qos.Latency.LTE(math.LegacyZeroDec()) ||
		qos.Sync.LTE(math.LegacyZeroDec()) {
		return math.LegacyZeroDec(), fmt.Errorf("QoS excellence scores is below 0")
	}
	return qos.Availability.Quo(qos.Sync).Quo(qos.Latency).ApproxRoot(3)
}
