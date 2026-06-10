package specfetcher

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockRoundTripper serves canned responses keyed by exact request URL, so the GitHub fetch path
// can be exercised end-to-end without network access (via Config.HTTPClient).
type mockRoundTripper struct {
	responses map[string]string
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	body, ok := m.responses[req.URL.String()]
	status := http.StatusOK
	if !ok {
		body, status = "not found", http.StatusNotFound
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}, nil
}

func newMockFetcher(responses map[string]string) *Fetcher {
	return New(Config{HTTPClient: &http.Client{Transport: &mockRoundTripper{responses: responses}}})
}

// TestFetchAllRawFiles_GitHub verifies the shared raw-fetch path: list a directory via the
// contents API and download each .json file's raw bytes, ignoring non-json and directories.
func TestFetchAllRawFiles_GitHub(t *testing.T) {
	listing := `[{"name":"a.json","type":"file"},{"name":"b.json","type":"file"},{"name":"readme.md","type":"file"},{"name":"sub","type":"dir"}]`
	aBody := `{"providers":[{"address":"provider0","chains":["ETH1"]}]}`
	bBody := `{"anything":"goes"}`
	f := newMockFetcher(map[string]string{
		"https://api.github.com/repos/owner/repo/contents/dir?ref=main": listing,
		"https://raw.githubusercontent.com/owner/repo/main/dir/a.json":  aBody,
		"https://raw.githubusercontent.com/owner/repo/main/dir/b.json":  bBody,
	})

	files, err := f.FetchAllRawFiles(context.Background(), "https://github.com/owner/repo/tree/main/dir")
	require.NoError(t, err)
	require.Len(t, files, 2) // only the two .json files of type "file"

	gotBodies := map[string]bool{}
	for _, content := range files {
		gotBodies[string(content)] = true
	}
	require.True(t, gotBodies[aBody])
	require.True(t, gotBodies[bBody])
}

// TestFetchAllRawFiles_PartialFailureTolerantVsStrict pins the two opposite contracts that share
// this fetch path. The directory lists two files but only a.json is served (b.json -> 404), so the
// fetch is partial. FetchAllRawFiles (spec default) tolerates it and returns the one file that
// succeeded; the strict mode used by the whitelist (Config.FailOnPartial) instead fails the whole
// fetch so the caller can keep its last-known-good snapshot rather than swap in a partial set.
func TestFetchAllRawFiles_PartialFailureTolerantVsStrict(t *testing.T) {
	const repoURL = "https://github.com/owner/repo/tree/main/dir"
	listing := `[{"name":"a.json","type":"file"},{"name":"b.json","type":"file"}]`
	aBody := `{"providers":[{"address":"provider0","chains":["ETH1"]}]}`
	responses := map[string]string{
		"https://api.github.com/repos/owner/repo/contents/dir?ref=main": listing,
		"https://raw.githubusercontent.com/owner/repo/main/dir/a.json":  aBody,
		// b.json is intentionally absent -> the mock returns 404 for it (a per-file fetch failure).
	}

	// Tolerant (spec default): partial failure still returns the files that succeeded.
	tolerant := newMockFetcher(responses)
	files, err := tolerant.FetchAllRawFiles(context.Background(), repoURL)
	require.NoError(t, err)
	require.Len(t, files, 1)

	// Strict (whitelist): any per-file failure fails the whole fetch.
	strictConfig := Config{HTTPClient: &http.Client{Transport: &mockRoundTripper{responses: responses}}, FailOnPartial: true}
	strict := New(strictConfig)
	_, err = strict.FetchAllRawFiles(context.Background(), repoURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "partial fetch failure")
}

// TestFetchAllSpecs_GitHubStillWorks guards the behavior-preserving refactor: FetchAllSpecs now
// layers on FetchAllRawFiles, and must still parse the fetched files into specs keyed by index.
func TestFetchAllSpecs_GitHubStillWorks(t *testing.T) {
	listing := `[{"name":"eth.json","type":"file"}]`
	specBody := `{"proposal":{"specs":[{"index":"ETH1"}]}}`
	f := newMockFetcher(map[string]string{
		"https://api.github.com/repos/owner/repo/contents/dir?ref=main":  listing,
		"https://raw.githubusercontent.com/owner/repo/main/dir/eth.json": specBody,
	})

	specs, err := f.FetchAllSpecs(context.Background(), "https://github.com/owner/repo/tree/main/dir")
	require.NoError(t, err)
	require.Contains(t, specs, "ETH1")
}
