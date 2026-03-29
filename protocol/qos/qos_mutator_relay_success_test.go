package qos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateAvailabilityScore(t *testing.T) {
	avialabilityAsFloat := AvailabilityPercentage
	precision := uint64(10000)

	qosReport := QoSReport{}
	qosReport.totalRelays = precision
	qosReport.answeredRelays = precision - uint64(avialabilityAsFloat*float64(precision))
	qoSMutatorRelaySuccess := QoSMutatorRelaySuccess{}
	downTime, availabilityScore := qoSMutatorRelaySuccess.calculateAvailabilityScore(&qosReport)
	require.Equal(t, downTime, avialabilityAsFloat)
	require.Equal(t, availabilityScore, 0.0)

	qosReport.totalRelays = 2 * precision
	qosReport.answeredRelays = 2*precision - uint64(avialabilityAsFloat*float64(precision))
	downTime, availabilityScore = qoSMutatorRelaySuccess.calculateAvailabilityScore(&qosReport)
	require.Equal(t, downTime*2, avialabilityAsFloat)
	require.InDelta(t, availabilityScore, 0.5, 1e-9)
}
