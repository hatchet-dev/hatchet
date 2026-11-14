package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TenantControl is 1 for each operator control set on a tenant, as this process last read
	// them. control is "paused" or "ingest". A series is removed when its control is cleared, so
	// the metric is empty unless an operator has acted.
	TenantControl = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hatchet_tenant_control",
		Help: "Operator controls set on a tenant (1 while set)",
	}, []string{"tenant_id", "control"})

	// TenantControlRefused counts the new runs and events this process refused because of an
	// operator control. Its series exist only while the tenant's control is set.
	TenantControlRefused = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hatchet_tenant_control_refused_total",
		Help: "New runs and events refused because of an operator control on the tenant",
	}, []string{"tenant_id", "control"})
)
