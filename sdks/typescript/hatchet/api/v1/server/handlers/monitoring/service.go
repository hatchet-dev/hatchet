package monitoring

import (
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type MonitoringService struct {
	enabled          bool
	permittedTenants []string
	eventName        string
	workflowName     string
	probeTimeout     time.Duration
	config           *server.ServerConfig
	l                *zerolog.Logger
	tlsRootCAFile    string
}

func NewMonitoringService(config *server.ServerConfig) *MonitoringService {
	return &MonitoringService{
		enabled:          config.Runtime.Monitoring.Enabled,
		l:                config.Logger,
		permittedTenants: config.Runtime.Monitoring.PermittedTenants,
		eventName:        "monitoring:probe",
		workflowName:     "probe-workflow",
		probeTimeout:     config.Runtime.Monitoring.ProbeTimeout,
		tlsRootCAFile:    config.Runtime.Monitoring.TLSRootCAFile,
		config:           config,
	}
}
