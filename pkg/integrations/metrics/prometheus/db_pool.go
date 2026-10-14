package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// FairpoolWait records how long an acquire waited for a connection slot.
	// It is observed only when the slot was not immediately available.
	FairpoolWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "hatchet_fairpool_wait_seconds",
		Help:    "Time an acquire waited for a fairpool connection slot",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"pool", "bucket", "outcome"})

	// FairpoolRejections counts acquires that gave up after MaxWait.
	// bucket is "tenant" or "shared". The series is not labeled by tenant id: a rejection
	// counter would otherwise grow one series per tenant for the life of the process.
	FairpoolRejections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_fairpool_rejections_total",
		Help: "Database connection acquires rejected because a fairpool bucket was at its cap",
	}, []string{"pool", "bucket"})

	// FairpoolHeldConns is the number of connections a bucket currently holds.
	// tenant_id is the tenant id, or "shared". Series are removed when the count returns to zero.
	FairpoolHeldConns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hatchet_fairpool_held_connections",
		Help: "Database connections currently held by a fairpool bucket",
	}, []string{"pool", "tenant_id"})
)
