package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DurableTaskBudgetWait records how long a DurableTask stream paused reading to wait for in-flight
// budget. Requests that get budget straight away are not observed. budget is "stream" or "tenant";
// outcome is "acquired", or "canceled" if the stream ended while waiting. There is no tenant id
// label, to bound the series count; the sampled warning logged when a wait starts carries it.
var DurableTaskBudgetWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "hatchet_durable_task_budget_wait_seconds",
	Help:    "Time a DurableTask stream waited for its in-flight request budget",
	Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 300},
}, []string{"budget", "outcome"})
