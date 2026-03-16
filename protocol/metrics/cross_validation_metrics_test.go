package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

var (
	cvLabels         = []string{"spec", "apiInterface", "method"}
	cvProviderLabels = []string{"spec", "apiInterface", "method", "provider_address"}
)

func newConsumerForCVTest() *ConsumerMetricsManager {
	return &ConsumerMetricsManager{
		crossValidationRequestsTotalMetric:              prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cv_req"}, cvLabels),
		crossValidationSuccessTotalMetric:               prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cv_success"}, cvLabels),
		crossValidationFailedTotalMetric:                prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cv_failed"}, cvLabels),
		crossValidationProviderAgreementsTotalMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cv_agreements"}, cvProviderLabels),
		crossValidationProviderDisagreementsTotalMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_c_cv_disagreements"}, cvProviderLabels),
	}
}

func newSmartRouterForCVTest() *SmartRouterMetricsManager {
	return &SmartRouterMetricsManager{
		crossValidationRequestsTotalMetric:              prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_cv_req"}, cvLabels),
		crossValidationSuccessTotalMetric:               prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_cv_success"}, cvLabels),
		crossValidationFailedTotalMetric:                prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_cv_failed"}, cvLabels),
		crossValidationProviderAgreementsTotalMetric:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_cv_agreements"}, cvProviderLabels),
		crossValidationProviderDisagreementsTotalMetric: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "t_sr_cv_disagreements"}, cvProviderLabels),
		urlToProviderName:                               make(map[string]string),
	}
}

// cvRunner is a thin adapter that lets the shared test case table drive both
// ConsumerMetricsManager and SmartRouterMetricsManager cross-validation logic.
type cvRunner struct {
	invoke        func(chainId, apiInterface, method string, success bool, agreeing, disagreeing []string)
	total         *prometheus.CounterVec
	successC      *prometheus.CounterVec
	failedC       *prometheus.CounterVec
	agreements    *prometheus.CounterVec
	disagreements *prometheus.CounterVec
}

func newConsumerCVRunner() *cvRunner {
	cmm := newConsumerForCVTest()
	return &cvRunner{
		invoke:        cmm.SetCrossValidationMetric,
		total:         cmm.crossValidationRequestsTotalMetric,
		successC:      cmm.crossValidationSuccessTotalMetric,
		failedC:       cmm.crossValidationFailedTotalMetric,
		agreements:    cmm.crossValidationProviderAgreementsTotalMetric,
		disagreements: cmm.crossValidationProviderDisagreementsTotalMetric,
	}
}

func newSmartRouterCVRunner() *cvRunner {
	m := newSmartRouterForCVTest()
	return &cvRunner{
		invoke:        m.SetCrossValidationMetric,
		total:         m.crossValidationRequestsTotalMetric,
		successC:      m.crossValidationSuccessTotalMetric,
		failedC:       m.crossValidationFailedTotalMetric,
		agreements:    m.crossValidationProviderAgreementsTotalMetric,
		disagreements: m.crossValidationProviderDisagreementsTotalMetric,
	}
}

// TestSetCrossValidationMetric covers the cross-validation counter logic for
// both ConsumerMetricsManager and SmartRouterMetricsManager using the same
// table of cases.
func TestSetCrossValidationMetric(t *testing.T) {
	type cvCall struct {
		success               bool
		agreeing, disagreeing []string
	}

	cases := []struct {
		name              string
		calls             []cvCall
		wantTotal         float64
		wantSuccess       float64
		wantFailed        float64
		wantAgreements    map[string]float64
		wantDisagreements map[string]float64
	}{
		{
			name:      "success with agreeing and disagreeing providers",
			calls:     []cvCall{{success: true, agreeing: []string{"prov-A", "prov-B"}, disagreeing: []string{"prov-C"}}},
			wantTotal: 1, wantSuccess: 1,
			wantAgreements:    map[string]float64{"prov-A": 1, "prov-B": 1},
			wantDisagreements: map[string]float64{"prov-C": 1},
		},
		{
			name:      "failure with disagreeing providers",
			calls:     []cvCall{{success: false, disagreeing: []string{"prov-X", "prov-Y"}}},
			wantTotal: 1, wantFailed: 1,
			wantDisagreements: map[string]float64{"prov-X": 1, "prov-Y": 1},
		},
		{
			name:      "failure with no providers",
			calls:     []cvCall{{success: false}},
			wantTotal: 1, wantFailed: 1,
		},
		{
			// Verifies both that the request counter accumulates across calls and
			// that success+failed==total (partition invariant).
			name: "multi-call accumulates; success+failed==total",
			calls: []cvCall{
				{success: true, agreeing: []string{"prov-A"}},
				{success: false, disagreeing: []string{"prov-B"}},
				{success: true, agreeing: []string{"prov-A"}},
			},
			wantTotal: 3, wantSuccess: 2, wantFailed: 1,
		},
	}

	runners := []struct {
		name string
		new  func() *cvRunner
	}{
		{"consumer", newConsumerCVRunner},
		{"smart_router", newSmartRouterCVRunner},
	}

	const (
		spec   = "ETH1"
		api    = "jsonrpc"
		method = "eth_blockNumber"
	)

	for _, r := range runners {
		t.Run(r.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					runner := r.new()

					for _, c := range tc.calls {
						runner.invoke(spec, api, method, c.success, c.agreeing, c.disagreeing)
					}

					l := []string{spec, api, method}
					require.Equal(t, tc.wantTotal, testutil.ToFloat64(runner.total.WithLabelValues(l...)), "total")
					require.Equal(t, tc.wantSuccess, testutil.ToFloat64(runner.successC.WithLabelValues(l...)), "success")
					require.Equal(t, tc.wantFailed, testutil.ToFloat64(runner.failedC.WithLabelValues(l...)), "failed")

					for prov, want := range tc.wantAgreements {
						require.Equal(t, want,
							testutil.ToFloat64(runner.agreements.WithLabelValues(spec, api, method, prov)),
							"agreement[%s]", prov)
					}
					for prov, want := range tc.wantDisagreements {
						require.Equal(t, want,
							testutil.ToFloat64(runner.disagreements.WithLabelValues(spec, api, method, prov)),
							"disagreement[%s]", prov)
					}
				})
			}
		})
	}
}

func TestConsumerSetCrossValidationMetric_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	})
}

func TestSmartRouterSetCrossValidationMetric_NilManager(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	})
}
