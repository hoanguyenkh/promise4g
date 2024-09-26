package promise4g

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promisesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "promises_created_total",
		Help: "The total number of promises created",
	})

	promiseExecutionTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "promise_execution_time_seconds",
		Help:    "The execution time of promises in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
	})

	concurrentPromises = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "concurrent_promises",
		Help: "The number of promises currently executing",
	})
)

func incrementPromisesCreated() {
	promisesCreated.Inc()
}

func observePromiseExecutionTime(seconds float64) {
	promiseExecutionTime.Observe(seconds)
}

func incrementConcurrentPromises() {
	concurrentPromises.Inc()
}

func decrementConcurrentPromises() {
	concurrentPromises.Dec()
}
