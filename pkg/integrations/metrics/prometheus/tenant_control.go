package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TenantControl is 1 for each operator control set on a tenant, as this process last read them.
// control is "paused", "slow", "budget" or "ingest". Series are removed when the control is cleared, so the metric is
// empty unless an operator has acted.
var TenantControl = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "hatchet_tenant_control",
	Help: "Operator controls set on a tenant (1 while set)",
}, []string{"tenant_id", "control"})
