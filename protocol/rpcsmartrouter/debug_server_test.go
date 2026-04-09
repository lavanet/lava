package rpcsmartrouter

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

func newEmptyOptimizersRouter() *common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer] {
	return &common.SafeSyncMap[string, *provideroptimizer.ProviderOptimizer]{}
}

func postTimeWarpRouter(mux http.Handler, rawBody string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/debug/time-warp", strings.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

// TestDebugTimeWarp_SmartRouter_OffsetBoundaryValidation mirrors the rpcconsumer boundary
// test for rpcsmartrouter — the handler is a verbatim copy and must enforce the same ceiling.
func TestDebugTimeWarp_SmartRouter_OffsetBoundaryValidation(t *testing.T) {
	cases := []struct {
		name           string
		rawBody        string
		wantStatus     int
		wantBodySubstr string
	}{
		{name: "negative", rawBody: `{"offset_seconds":-1}`, wantStatus: http.StatusBadRequest, wantBodySubstr: ">= 0"},
		{name: "NaN_literal_bad_json", rawBody: `{"offset_seconds":NaN}`, wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON"},
		{name: "pos_inf_via_overflow", rawBody: `{"offset_seconds":1e999}`, wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON"},
		{name: "neg_inf_via_overflow", rawBody: `{"offset_seconds":-1e999}`, wantStatus: http.StatusBadRequest, wantBodySubstr: "invalid JSON"},
		{name: "one_over_new_ceiling_86401", rawBody: `{"offset_seconds":86401}`, wantStatus: http.StatusBadRequest, wantBodySubstr: "86400"},
		{name: "doc_example_90000", rawBody: `{"offset_seconds":90000}`, wantStatus: http.StatusBadRequest, wantBodySubstr: "86400"},
		{name: "zero_resets_warp", rawBody: `{"offset_seconds":0}`, wantStatus: http.StatusOK},
		{name: "one_hour_3600", rawBody: `{"offset_seconds":3600}`, wantStatus: http.StatusOK},
		{name: "old_ceiling_86340", rawBody: `{"offset_seconds":86340}`, wantStatus: http.StatusOK},
		// *** regression: was HTTP 400 with old code, must now be 200 ***
		{name: "just_above_old_ceiling_86341", rawBody: `{"offset_seconds":86341}`, wantStatus: http.StatusOK},
		{name: "new_ceiling_86400", rawBody: `{"offset_seconds":86400}`, wantStatus: http.StatusOK},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var offsetNano atomic.Int64
			mux := buildDebugMux(newEmptyOptimizersRouter(), &offsetNano)

			rr := postTimeWarpRouter(mux, tc.rawBody)

			require.Equal(t, tc.wantStatus, rr.Code,
				"body=%q response=%q", tc.rawBody, rr.Body.String())
			if tc.wantBodySubstr != "" {
				require.Contains(t, rr.Body.String(), tc.wantBodySubstr)
			}
		})
	}
}

// TestDebugTimeWarp_SmartRouter_ErrorMessageContainsNewCeiling mirrors the ceiling-message test.
func TestDebugTimeWarp_SmartRouter_ErrorMessageContainsNewCeiling(t *testing.T) {
	var offsetNano atomic.Int64
	mux := buildDebugMux(newEmptyOptimizersRouter(), &offsetNano)

	rr := postTimeWarpRouter(mux, `{"offset_seconds":86401}`)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "86400")
	require.NotContains(t, rr.Body.String(), "86340")
	require.Contains(t, rr.Body.String(), "24h")
}
