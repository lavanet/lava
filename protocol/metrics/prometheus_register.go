package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// registerOrReuse registers a Prometheus collector, returning the existing
// collector on AlreadyRegisteredError so callers always hold the instance
// that Prometheus actually exposes.
func registerOrReuse[T prometheus.Collector](c T) T {
	if err := prometheus.Register(c); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			if existing, ok := are.ExistingCollector.(T); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}
