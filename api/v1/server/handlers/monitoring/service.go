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
	probeTimeout     time.Duration

	l *zerolog.Logger
}

func NewMonitoringService(config *server.ServerConfig) *MonitoringService {
	return &MonitoringService{
		enabled:          config.Runtime.Monitoring.Enabled,
		l:                config.Logger,
		permittedTenants: config.Runtime.Monitoring.PermittedTenants,
		eventName:        "monitoring:probe",
		probeTimeout:     config.Runtime.Monitoring.ProbeTimeout,
	}
}
