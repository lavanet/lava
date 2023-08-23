package upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHappyFlow(t *testing.T) {
	protocolVersion := ProtocolVersion{
		ConsumerVersion: "0.22.5",
		ProviderVersion: "0.22.5",
	}
	require.False(t, HasVersionMismatch("0.22.5", protocolVersion.ConsumerVersion))
	require.False(t, HasVersionMismatch("0.21.0", protocolVersion.ConsumerVersion))
	require.False(t, HasVersionMismatch("0.21.5", protocolVersion.ConsumerVersion))
	require.False(t, HasVersionMismatch("0.21.0", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("0.23.0", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("0.22.6", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("1.22.0", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("1.22.1", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("1.23.1", protocolVersion.ConsumerVersion))
	require.True(t, HasVersionMismatch("0.22.5.1", protocolVersion.ConsumerVersion))
	require.False(t, HasVersionMismatch("0.22.0", "0.22.0.1"))
}
