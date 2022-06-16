package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (qos *QualityOfServiceReport) ComputeQoS() (sdk.Dec, error) {
	if qos.Availability.GT(sdk.OneDec()) || qos.Availability.GT(sdk.ZeroDec()) ||
		qos.Latency.GT(sdk.OneDec()) || qos.Latency.GT(sdk.ZeroDec()) ||
		qos.Sync.GT(sdk.OneDec()) || qos.Sync.GT(sdk.ZeroDec()) {
		return sdk.ZeroDec(), fmt.Errorf("QoS scores is not between 0-1")
	}
	return qos.Availability.Mul(qos.Sync).Mul(qos.Latency).ApproxSqrt()
}
