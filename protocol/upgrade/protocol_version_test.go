package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHappyFlow(t *testing.T) {
	version := "0.22.5"

	require.False(t, HasVersionMismatch("0.22.5", version))
	require.False(t, HasVersionMismatch("0.21.5", version))
	require.False(t, HasVersionMismatch("0.21.0", version))

	require.True(t, HasVersionMismatch("0.22.5.1", version))
	require.True(t, HasVersionMismatch("0.22.6", version))
	require.True(t, HasVersionMismatch("0.23.0", version))
	require.True(t, HasVersionMismatch("1.22.0", version))
	require.True(t, HasVersionMismatch("1.22.1", version))
	require.True(t, HasVersionMismatch("1.23.1", version))

	require.False(t, HasVersionMismatch("0.22.0", "0.22.0.1"))
}
