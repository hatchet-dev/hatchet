package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DurableTaskBudgetWait records how long a DurableTask stream stopped reading because its in-flight
// budget was full. It is observed only when the budget was not immediately available. budget is
// "stream" or "tenant"; the series is not labeled by tenant id, which the wait log carries instead.
var DurableTaskBudgetWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "hatchet_durable_task_budget_wait_seconds",
	Help:    "Time a DurableTask stream waited for its in-flight request budget",
	Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10, 30, 60, 300},
}, []string{"budget", "outcome"})
