package rpcclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConcurrentRequestsCarryOwnHeaders is the regression test for the
// traceparent-race fix: under concurrent CallContext calls on a shared
// client, each request must receive only the headers attached to its own
// ctx, with no leakage from concurrent calls.
func TestConcurrentRequestsCarryOwnHeaders(t *testing.T) {
	const N = 50

	// Collect traceparent headers seen by the server, one entry per request.
	var (
		mu         sync.Mutex
		gotHeaders []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tp := r.Header.Get("traceparent")
		mu.Lock()
		gotHeaders = append(gotHeaders, tp)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"ok"}`))
	}))
	defer srv.Close()

	client, err := DialHTTP(srv.URL)
	require.NoError(t, err)
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(idx int) {
			defer wg.Done()
			h := http.Header{}
			tp := fmt.Sprintf("00-00000000000000000000000000%06d-1234567890abcdef-01", idx)
			h.Set("traceparent", tp)
			ctx := WithHeaders(context.Background(), h)
			_, _ = client.CallContext(ctx, []byte("1"), "test_method", []interface{}{idx}, true, false)
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, gotHeaders, N, "expected %d requests to reach the server", N)

	// Every traceparent must be distinct — if headers leaked across goroutines,
	// we'd see duplicates or mismatched values.
	distinct := make(map[string]struct{}, N)
	for _, tp := range gotHeaders {
		distinct[tp] = struct{}{}
	}
	require.Equal(t, N, len(distinct),
		"concurrent requests leaked headers — saw %d requests but only %d distinct traceparents",
		N, len(distinct))
}
