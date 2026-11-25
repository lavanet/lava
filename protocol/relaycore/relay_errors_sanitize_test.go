package relaycore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelayErrors_sanitizeError(t *testing.T) {
	cases := []struct {
		desc     string
		input    string
		expected string
	}{
		{"masks session id", "relay failed SessionId:1867231 upstream", "relay failed SessionId:* upstream"},
		{"masks guid", "relay failed GUID:9817236406184312 upstream", "relay failed GUID:* upstream"},
		{"masks both in one message", "SessionId:12 GUID:34 timeout", "SessionId:* GUID:* timeout"},
		{"leaves other numbers alone", "status code 504 from provider 3", "status code 504 from provider 3"},
		{"leaves plain errors untouched", "connection refused", "connection refused"},
		{"masks repeated occurrences", "SessionId:1 then SessionId:2", "SessionId:* then SessionId:*"},
		{"handles empty message", "", ""},
	}

	for _, tc := range cases {
		got := (&RelayErrors{}).sanitizeError(errors.New(tc.input))
		require.Equal(t, tc.expected, got, "%s, input %q", tc.desc, tc.input)
	}
}
