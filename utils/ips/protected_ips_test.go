package ips

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidAddressesProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value string
		valid bool
	}{
		{valid: false, value: ":"},
		{valid: false, value: "http://::"},
		{valid: false, value: "::"},
		{valid: false, value: "http://localhost:1211"},
		{valid: false, value: "http://127.0.0.1:1211"},
		{valid: false, value: "127.0.0.1:3123"},
		{valid: false, value: "127.0.0.126:3123"},
		{valid: false, value: "http://127.0.0.126:3123"},
		{valid: false, value: "localhost:1211"},
		{valid: false, value: "0.0.0.0:1211"},
		{valid: false, value: "http://0.0.0.0:1211"},
		{valid: false, value: "http://0.0.0.0:1211"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			value := IsValidNetworkAddress(tt.value)
			require.Equal(t, tt.valid, value)

			value = ValidateProviderAddressConsensus(tt.value)
			require.Equal(t, tt.valid, value)
		})
	}
}
