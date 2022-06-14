package types

import sdk "github.com/cosmos/cosmos-sdk/types"

func (qos *QualityOfServiceReport) ComputeQoS() (sdk.Dec, error) {
	return qos.Availability.Mul(qos.Freshness).Mul(qos.Latency).ApproxSqrt()
}
