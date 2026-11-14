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

	// FairpoolHeldSeconds is the wall-clock time a bucket's checkouts were held, added when each is
	// released. It includes lock waits and the engine's own work inside a transaction, so it is a
	// proxy for database load. tenant_id is the tenant id, or "shared". A series is removed once its
	// bucket has had no checkout for ten minutes.
	FairpoolHeldSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_fairpool_held_seconds_total",
		Help: "Seconds database connections were checked out by a fairpool bucket",
	}, []string{"pool", "tenant_id"})

	// FairpoolBudgetWait records how long a per-tenant loop waited for its tenant's budget at its
	// yield point (fairpool.Pool.AwaitBudget). It is observed only when the loop waited.
	FairpoolBudgetWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "hatchet_fairpool_budget_wait_seconds",
		Help:    "Time a per-tenant scheduler or controller loop waited for its tenant's database time budget",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"pool"})

	// FairpoolConnectionLimit is the cap of a bucket that has one. For bucket "tenant" the cap applies
	// to each tenant separately.
	FairpoolConnectionLimit = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hatchet_fairpool_connection_limit",
		Help: "Database checkouts a fairpool bucket may have in flight at once",
	}, []string{"pool", "bucket"})
)
