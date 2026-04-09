package rpcconsumer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/provideroptimizer"
	"github.com/stretchr/testify/require"
)

// newEmptyOptimizers returns an empty SafeSyncMap suitable for wiring into buildDebugMux
// when the test only needs to exercise the validation guards (before the optimizer loop).
func newEmptyOptimizers() *common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer] {
	return &common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer]{}
}

// postTimeWarp sends a POST /debug/time-warp request with the given raw JSON body and
// returns the recorder so the caller can inspect the status code and body.
func postTimeWarp(mux http.Handler, rawBody string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/debug/time-warp", strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// TestDebugTimeWarp_OffsetBoundaryValidation verifies that the /debug/time-warp handler
// accepts and rejects offset values exactly as specified in the plan.
//
// Primary regression targets (changed by this PR):
//   - 86341: was rejected with old ceiling (86340), must now be accepted.
//   - 86400: new exact ceiling, must be accepted (operator is >, not >=).
//
// The test calls buildDebugMux directly so it exercises the real production handler,
// not a copy of it.
func TestDebugTimeWarp_OffsetBoundaryValidation(t *testing.T) {
	cases := []struct {
		name           string
		rawBody        string
		wantStatus     int
		wantBodySubstr string
	}{
		// ── rejection cases ──────────────────────────────────────────────────────
		{
			// negative offset — backward shift freezes the optimizer
			name: "negative", rawBody: `{"offset_seconds":-1}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: ">= 0",
		},
		{
			// NaN is invalid JSON syntax — decoder rejects it before the finite guard fires
			name: "NaN_literal_bad_json", rawBody: `{"offset_seconds":NaN}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON",
		},
		{
			// 1e999 overflows float64 — Go's JSON decoder rejects it as invalid JSON
			// (not +Inf), so the "invalid JSON" branch fires, not the finite guard.
			name: "pos_inf_via_overflow", rawBody: `{"offset_seconds":1e999}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON",
		},
		{
			// same for -1e999
			name: "neg_inf_via_overflow", rawBody: `{"offset_seconds":-1e999}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON",
		},
		{
			// one second above the new 24 h ceiling
			name: "one_over_new_ceiling_86401", rawBody: `{"offset_seconds":86401}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: "86400",
		},
		{
			// the doc-example curl value that was already over the old 86340 ceiling
			name: "doc_example_90000", rawBody: `{"offset_seconds":90000}`,
			wantStatus: http.StatusBadRequest, wantBodySubstr: "86400",
		},
		// ── acceptance cases ─────────────────────────────────────────────────────
		{
			// zero always accepted — resets any active warp
			name: "zero_resets_warp", rawBody: `{"offset_seconds":0}`,
			wantStatus: http.StatusOK,
		},
		{
			// ordinary +1 h — well within limit
			name: "one_hour_3600", rawBody: `{"offset_seconds":3600}`,
			wantStatus: http.StatusOK,
		},
		{
			// old ceiling value — must still be accepted after raising the cap
			name: "old_ceiling_86340", rawBody: `{"offset_seconds":86340}`,
			wantStatus: http.StatusOK,
		},
		{
			// *** regression: was HTTP 400 with old code (ceiling was 86340), must now be 200 ***
			name: "just_above_old_ceiling_86341", rawBody: `{"offset_seconds":86341}`,
			wantStatus: http.StatusOK,
		},
		{
			// *** regression: was HTTP 400 with old code, must now be 200 (new exact ceiling) ***
			name: "new_ceiling_86400", rawBody: `{"offset_seconds":86400}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var offsetNano atomic.Int64
			mux := buildDebugMux(newEmptyOptimizers(), &offsetNano)

			rr := postTimeWarp(mux, tc.rawBody)

			require.Equal(t, tc.wantStatus, rr.Code,
				"body=%q response=%q", tc.rawBody, rr.Body.String())
			if tc.wantBodySubstr != "" {
				require.Contains(t, rr.Body.String(), tc.wantBodySubstr,
					"expected response to contain %q", tc.wantBodySubstr)
			}
		})
	}
}

// TestDebugTimeWarp_MethodNotAllowed verifies that GET on /debug/time-warp returns 405.
func TestDebugTimeWarp_MethodNotAllowed(t *testing.T) {
	var offsetNano atomic.Int64
	mux := buildDebugMux(newEmptyOptimizers(), &offsetNano)

	req := httptest.NewRequest(http.MethodGet, "/debug/time-warp", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

// TestDebugTimeWarp_InvalidJSON verifies that malformed JSON returns 400 with "invalid JSON".
func TestDebugTimeWarp_InvalidJSON(t *testing.T) {
	var offsetNano atomic.Int64
	mux := buildDebugMux(newEmptyOptimizers(), &offsetNano)

	rr := postTimeWarp(mux, `{bad json}`)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "invalid JSON")
}

// TestDebugTime_ReturnsJSON verifies that GET /debug/time returns a JSON object with
// the three expected fields.
func TestDebugTime_ReturnsJSON(t *testing.T) {
	var offsetNano atomic.Int64
	mux := buildDebugMux(newEmptyOptimizers(), &offsetNano)

	req := httptest.NewRequest(http.MethodGet, "/debug/time", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	require.Contains(t, body, `"real_time"`)
	require.Contains(t, body, `"effective_time"`)
	require.Contains(t, body, `"offset_seconds"`)
}

// TestDebugTimeWarp_ErrorMessageContainsNewCeiling verifies that when an over-limit value
// is rejected, the error message reports 86400 (not the old 86340).
func TestDebugTimeWarp_ErrorMessageContainsNewCeiling(t *testing.T) {
	var offsetNano atomic.Int64
	mux := buildDebugMux(newEmptyOptimizers(), &offsetNano)

	rr := postTimeWarp(mux, `{"offset_seconds":86401}`)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "86400",
		"error message must advertise the new 24 h ceiling (86400), not the old 86340")
	require.NotContains(t, rr.Body.String(), "86340",
		"old ceiling value must not appear in the error message after this PR")
	require.Contains(t, rr.Body.String(), "24h")
}

