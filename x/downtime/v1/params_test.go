package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParams(t *testing.T) {
	require.NoError(t, validateDowntimeDuration(10*time.Second))
	require.ErrorContains(t, validateDowntimeDuration(10), "invalid parameter type")
	require.ErrorContains(t, validateDowntimeDuration(time.Duration(-10)), "invalid downtime duration")
}
