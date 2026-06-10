package rpcconsumer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
	"github.com/stretchr/testify/require"
)

const geoEU = uint64(planstypes.Geolocation_EU) // 2

func writeTempWhitelist(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "whitelist.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestProviderWhitelistFetcher_LoadFromLocalFile(t *testing.T) {
	path := writeTempWhitelist(t, `{"chains":{"ETH1":{"EU":["provider0"]},"LAV1":{"EU":["provider0"]}}}`)
	pw := lavasession.NewProviderWhitelist()
	fetcher := NewProviderWhitelistFetcher(path, "", "", "", time.Hour, pw)

	require.True(t, fetcher.loadOnce(context.Background()))
	require.True(t, pw.Enabled())
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
	require.True(t, pw.IsAllowed("LAV1", geoEU, "provider0"))
	require.False(t, pw.IsAllowed("ETH1", geoEU, "provider1"))
}

func TestProviderWhitelistFetcher_MalformedRefreshKeepsPrevious(t *testing.T) {
	path := writeTempWhitelist(t, `{"chains":{"ETH1":{"EU":["provider0"]}}}`)
	pw := lavasession.NewProviderWhitelist()
	fetcher := NewProviderWhitelistFetcher(path, "", "", "", time.Hour, pw)

	require.True(t, fetcher.loadOnce(context.Background()))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))

	// Overwrite the source with malformed content; the refresh fails and keeps the last-good list.
	require.NoError(t, os.WriteFile(path, []byte(`{ not valid json `), 0o600))
	require.False(t, fetcher.loadOnce(context.Background()))
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelistFetcher_MissingFileKeepsPassthrough(t *testing.T) {
	pw := lavasession.NewProviderWhitelist()
	fetcher := NewProviderWhitelistFetcher(filepath.Join(t.TempDir(), "does-not-exist.json"), "", "", "", time.Hour, pw)

	require.False(t, fetcher.loadOnce(context.Background()))
	// Never loaded -> stays in passthrough.
	require.False(t, pw.Enabled())
	require.True(t, pw.IsAllowed("ETH1", geoEU, "provider0"))
}

func TestProviderWhitelistFetcher_ResolveTokenForSource(t *testing.T) {
	const (
		ghURL = "https://github.com/owner/repo/tree/main/dir"
		glURL = "https://gitlab.com/owner/repo/-/tree/main/dir"
		local = "/tmp/whitelist.json"
	)

	// A dedicated whitelist token wins regardless of the source's provider.
	require.Equal(t, "dedicated", NewProviderWhitelistFetcher(ghURL, "dedicated", "gh", "gl", time.Hour, nil).resolveTokenForSource())
	require.Equal(t, "dedicated", NewProviderWhitelistFetcher(glURL, "dedicated", "gh", "gl", time.Hour, nil).resolveTokenForSource())

	// With no dedicated token, fall back to the provider-specific token.
	require.Equal(t, "gh", NewProviderWhitelistFetcher(ghURL, "", "gh", "gl", time.Hour, nil).resolveTokenForSource())
	require.Equal(t, "gl", NewProviderWhitelistFetcher(glURL, "", "gh", "gl", time.Hour, nil).resolveTokenForSource())

	// A local file source needs no token.
	require.Equal(t, "", NewProviderWhitelistFetcher(local, "", "gh", "gl", time.Hour, nil).resolveTokenForSource())
}
