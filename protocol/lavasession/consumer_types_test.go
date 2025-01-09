package lavasession

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/protocol/qos"
	"github.com/stretchr/testify/require"
)

func TestCalculateAvailabilityScore(t *testing.T) {
	avialabilityAsFloat, err := AvailabilityPercentage.Float64()
	require.NoError(t, err)
	precision := uint64(10000)
	qosManager := qos.NewQoSManager()
	qosManager.SetTotalRelays(precision)
	qosManager.SetAnsweredRelays(precision - uint64(avialabilityAsFloat*float64(precision)))
	downTime, availabilityScore := qosManager.CalculateAvailabilityScore()
	downTimeFloat, err := downTime.Float64()
	require.NoError(t, err)
	require.Equal(t, downTimeFloat, avialabilityAsFloat)
	require.Zero(t, availabilityScore.BigInt().Uint64())

	qosManager.SetTotalRelays(2 * precision)
	qosManager.SetAnsweredRelays(2*precision - uint64(avialabilityAsFloat*float64(precision)))
	downTime, availabilityScore = qosManager.CalculateAvailabilityScore()
	downTimeFloat, err = downTime.Float64()
	require.NoError(t, err)
	halfDec, err := sdk.NewDecFromStr("0.5")
	require.NoError(t, err)
	require.Equal(t, downTimeFloat*2, avialabilityAsFloat)
	require.Equal(t, halfDec, availabilityScore)
}
