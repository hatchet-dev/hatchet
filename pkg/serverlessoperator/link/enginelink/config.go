package enginelink

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

// LinkName labels the in-engine mode's metrics.
const LinkName = "engine"

// ConfigFromServer maps the engine's SERVER_SERVERLESS_OPERATOR_* settings onto the core's
// Config. The health server is disabled: the engine serves its own health and metrics
// endpoints, and the core's metrics are registered on the default Prometheus registry either
// way. Zero values fall through to the core's defaults.
func ConfigFromServer(cf server.ServerlessOperatorConfigFile) serverlessoperator.Config {
	return serverlessoperator.Config{
		OperatorName:           cf.OperatorName,
		LinkName:               LinkName,
		DefaultSlots:           cf.DefaultSlots,
		DurableSlots:           cf.DurableSlots,
		LeaseTTL:               cf.LeaseTTL,
		HeartbeatInterval:      cf.HeartbeatInterval,
		RebalanceInterval:      cf.RebalanceInterval,
		ShedHysteresis:         cf.ShedHysteresis,
		DrainTimeout:           cf.DrainTimeout,
		RoutingRefreshInterval: cf.RoutingRefreshInterval,
		HealthcheckTimeout:     cf.HealthcheckTimeout,
		HealthcheckConcurrency: cf.HealthcheckConcurrency,
		WSMaxFrameBytes:        cf.WSMaxFrameBytes,
		WSPingInterval:         cf.WSPingInterval,
		HealthPort:             0,
	}
}
