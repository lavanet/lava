package metrics

type AggregatedMetric struct {
	TotalLatency int64
	RelaysCount  int64
	SuccessCount int64
}

type MetricService struct {
	aggregatedMetricMap map[string]map[string]map[string]AggregatedMetric
	// channel chan Analytics

}
