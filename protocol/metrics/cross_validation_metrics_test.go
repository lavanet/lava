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

// ---- Consumer tests ----

func TestConsumerSetCrossValidationMetric_Success(t *testing.T) {
	cmm := newConsumerForCVTest()
	agreeing := []string{"prov-A", "prov-B"}
	disagreeing := []string{"prov-C"}

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, agreeing, disagreeing)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationProviderAgreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-A")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationProviderAgreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-B")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-C")))
}

func TestConsumerSetCrossValidationMetric_Failure(t *testing.T) {
	cmm := newConsumerForCVTest()
	disagreeing := []string{"prov-X", "prov-Y"}

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, disagreeing)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-X")))
	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-Y")))
}

func TestConsumerSetCrossValidationMetric_RequestCountsEveryCall(t *testing.T) {
	cmm := newConsumerForCVTest()

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, []string{"prov-B"})
	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)

	require.Equal(t, float64(3), testutil.ToFloat64(cmm.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(2), testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestConsumerSetCrossValidationMetric_NoProvidersOnFailure(t *testing.T) {
	cmm := newConsumerForCVTest()

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestConsumerSetCrossValidationMetric_FailedIncrementedOnFailure(t *testing.T) {
	cmm := newConsumerForCVTest()

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(cmm.crossValidationFailedTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestConsumerSetCrossValidationMetric_SuccessPlusFailedEqualsRequests(t *testing.T) {
	cmm := newConsumerForCVTest()

	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, []string{"prov-B"})
	cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)

	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}
	total := testutil.ToFloat64(cmm.crossValidationRequestsTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(cmm.crossValidationSuccessTotalMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(cmm.crossValidationFailedTotalMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestConsumerSetCrossValidationMetric_NilManager(t *testing.T) {
	var cmm *ConsumerMetricsManager
	require.NotPanics(t, func() {
		cmm.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	})
}

// ---- SmartRouter tests ----

func TestSmartRouterSetCrossValidationMetric_Success(t *testing.T) {
	m := newSmartRouterForCVTest()
	agreeing := []string{"prov-A", "prov-B"}
	disagreeing := []string{"prov-C"}

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, agreeing, disagreeing)

	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationProviderAgreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-A")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationProviderAgreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-B")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-C")))
}

func TestSmartRouterSetCrossValidationMetric_Failure(t *testing.T) {
	m := newSmartRouterForCVTest()
	disagreeing := []string{"prov-X", "prov-Y"}

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, disagreeing)

	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-X")))
	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationProviderDisagreementsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber", "prov-Y")))
}

func TestSmartRouterSetCrossValidationMetric_RequestCountsEveryCall(t *testing.T) {
	m := newSmartRouterForCVTest()

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, []string{"prov-B"})
	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)

	require.Equal(t, float64(3), testutil.ToFloat64(m.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(2), testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestSmartRouterSetCrossValidationMetric_NoProvidersOnFailure(t *testing.T) {
	m := newSmartRouterForCVTest()

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationRequestsTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestSmartRouterSetCrossValidationMetric_FailedIncrementedOnFailure(t *testing.T) {
	m := newSmartRouterForCVTest()

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, nil)

	require.Equal(t, float64(1), testutil.ToFloat64(m.crossValidationFailedTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
	require.Equal(t, float64(0), testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues("ETH1", "jsonrpc", "eth_blockNumber")))
}

func TestSmartRouterSetCrossValidationMetric_SuccessPlusFailedEqualsRequests(t *testing.T) {
	m := newSmartRouterForCVTest()

	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", false, nil, []string{"prov-B"})
	m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)

	labels := []string{"ETH1", "jsonrpc", "eth_blockNumber"}
	total := testutil.ToFloat64(m.crossValidationRequestsTotalMetric.WithLabelValues(labels...))
	success := testutil.ToFloat64(m.crossValidationSuccessTotalMetric.WithLabelValues(labels...))
	failed := testutil.ToFloat64(m.crossValidationFailedTotalMetric.WithLabelValues(labels...))

	require.Equal(t, float64(3), total)
	require.Equal(t, float64(2), success)
	require.Equal(t, float64(1), failed)
	require.Equal(t, total, success+failed)
}

func TestSmartRouterSetCrossValidationMetric_NilManager(t *testing.T) {
	var m *SmartRouterMetricsManager
	require.NotPanics(t, func() {
		m.SetCrossValidationMetric("ETH1", "jsonrpc", "eth_blockNumber", true, []string{"prov-A"}, nil)
	})
}
