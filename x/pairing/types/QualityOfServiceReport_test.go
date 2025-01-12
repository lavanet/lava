package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func createTestQosReportScores(forReputation bool) ([]math.LegacyDec, error) {
	qos1 := &QualityOfServiceReport{
		Latency:      sdk.MustNewDecFromStr("1.5"),
		Availability: sdk.MustNewDecFromStr("1"),
		Sync:         sdk.MustNewDecFromStr("0.1"),
	}
	qos2 := &QualityOfServiceReport{
		Latency:      sdk.MustNewDecFromStr("0.2"),
		Availability: sdk.MustNewDecFromStr("1"),
		Sync:         sdk.MustNewDecFromStr("0.1"),
	}
	qos3 := &QualityOfServiceReport{
		Latency:      sdk.MustNewDecFromStr("0.1"),
		Availability: sdk.MustNewDecFromStr("1"),
		Sync:         sdk.MustNewDecFromStr("0.5"),
	}
	qos4 := &QualityOfServiceReport{
		Latency:      sdk.MustNewDecFromStr("0.1"),
		Availability: sdk.MustNewDecFromStr("0.5"),
		Sync:         sdk.MustNewDecFromStr("0.5"),
	}

	res := []math.LegacyDec{}
	if forReputation {
		syncFactor := sdk.MustNewDecFromStr("0.5")
		qos1Res, errQos1 := qos1.ComputeQosExcellenceForReputation(syncFactor)
		if errQos1 != nil {
			return nil, errQos1
		}
		qos2Res, errQos2 := qos2.ComputeQosExcellenceForReputation(syncFactor)
		if errQos2 != nil {
			return nil, errQos2
		}
		qos3Res, errQos3 := qos3.ComputeQosExcellenceForReputation(syncFactor)
		if errQos3 != nil {
			return nil, errQos3
		}
		qos4Res, errQos4 := qos4.ComputeQosExcellenceForReputation(syncFactor)
		if errQos4 != nil {
			return nil, errQos4
		}
		res = append(res, qos1Res, qos2Res, qos3Res, qos4Res)
	} else {
		qos1Res, errQos1 := qos1.ComputeQoSExcellence()
		if errQos1 != nil {
			return nil, errQos1
		}
		qos2Res, errQos2 := qos2.ComputeQoSExcellence()
		if errQos2 != nil {
			return nil, errQos2
		}
		qos3Res, errQos3 := qos3.ComputeQoSExcellence()
		if errQos3 != nil {
			return nil, errQos3
		}
		qos4Res, errQos4 := qos4.ComputeQoSExcellence()
		if errQos4 != nil {
			return nil, errQos4
		}
		res = append(res, qos1Res, qos2Res, qos3Res, qos4Res)
	}

	return res, nil
}

func TestQosReport(t *testing.T) {
	res, err := createTestQosReportScores(false)
	require.NoError(t, err)
	require.True(t, res[0].LT(res[1]))
	require.True(t, res[0].LT(res[2]))
	require.True(t, res[0].LT(res[3]))

	require.True(t, res[1].GT(res[2]))
	require.True(t, res[1].GT(res[3]))

	require.True(t, res[3].LT(res[2]))
}

func TestQosReportForReputation(t *testing.T) {
	res, err := createTestQosReportScores(true)
	require.NoError(t, err)
	require.True(t, res[0].GT(res[1]))
	require.True(t, res[0].GT(res[2]))
	require.True(t, res[0].LT(res[3]))

	require.True(t, res[1].LT(res[2]))
	require.True(t, res[1].LT(res[3]))

	require.True(t, res[3].GT(res[2]))
}
