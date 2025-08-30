package qos

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateAvailabilityScore(t *testing.T) {
	avialabilityAsFloat, err := AvailabilityPercentage.Float64()
	require.NoError(t, err)
	precision := uint64(10000)

	qosReport := QoSReport{}
	qosReport.totalRelays = precision
	qosReport.answeredRelays = precision - uint64(avialabilityAsFloat*float64(precision))
	qoSMutatorRelaySuccess := QoSMutatorRelaySuccess{}
	downTime, availabilityScore := qoSMutatorRelaySuccess.calculateAvailabilityScore(&qosReport)
	downTimeFloat, err := downTime.Float64()
	require.NoError(t, err)
	require.Equal(t, downTimeFloat, avialabilityAsFloat)
	require.Zero(t, availabilityScore.BigInt().Uint64())

	qosReport.totalRelays = 2 * precision
	qosReport.answeredRelays = 2*precision - uint64(avialabilityAsFloat*float64(precision))
	downTime, availabilityScore = qoSMutatorRelaySuccess.calculateAvailabilityScore(&qosReport)
	downTimeFloat, err = downTime.Float64()
	require.NoError(t, err)
	halfDec, err := sdk.NewDecFromStr("0.5")
	require.NoError(t, err)
	require.Equal(t, downTimeFloat*2, avialabilityAsFloat)
	require.Equal(t, halfDec, availabilityScore)
}
