package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TenantPoolGateWait records how long a tenant waited for a connection slot.
	// It is observed only when the slot was not immediately available.
	TenantPoolGateWait = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "hatchet_db_pool_tenant_gate_wait_seconds",
		Help:    "Time a tenant waited for a database connection slot",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	}, []string{"pool", "outcome"})

	// TenantPoolGateRejections counts acquires that gave up after MaxWait.
	TenantPoolGateRejections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_db_pool_tenant_gate_rejections_total",
		Help: "Database connection acquires rejected because the tenant was at its pool cap",
	}, []string{"pool", "tenant_id"})

	// TenantPoolHeldConns is the number of connections a tenant currently holds.
	// Series are removed when the count returns to zero.
	TenantPoolHeldConns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hatchet_db_pool_tenant_held_conns",
		Help: "Database connections currently held by a tenant",
	}, []string{"pool", "tenant_id"})
)
