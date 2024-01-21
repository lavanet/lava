package badgeserver

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsService struct {
	TotalRequests      prometheus.Counter
	FailedRequests     prometheus.Counter
	SuccessfulRequests prometheus.Counter
}

func InitMetrics() *MetricsService {
	service := &MetricsService{}
	service.TotalRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "badges_total_request",
		Help: "The total request for a badge",
	})
	service.FailedRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "badges_failed_request",
		Help: "Number of failed request.",
	})
	service.SuccessfulRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "badges_success_request",
		Help: "Number of successful processed requests.",
	})

	return service
}

func (service *MetricsService) AddRequest(isSuccessful bool) {
	if service != nil {
		service.TotalRequests.Inc()
		if isSuccessful {
			service.SuccessfulRequests.Inc()
		} else {
			service.FailedRequests.Inc()
		}
	}
}
