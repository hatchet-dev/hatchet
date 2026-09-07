package enginelink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
)

func TestConfigFromServer(t *testing.T) {
	cf := server.ServerlessOperatorConfigFile{
		OperatorName:           "custom",
		DefaultSlots:           11,
		DurableSlots:           12,
		LeaseTTL:               13 * time.Second,
		HeartbeatInterval:      14 * time.Second,
		RebalanceInterval:      15 * time.Second,
		ShedHysteresis:         0.3,
		DrainTimeout:           16 * time.Second,
		RoutingRefreshInterval: 17 * time.Second,
		HealthcheckTimeout:     18 * time.Second,
		HealthcheckConcurrency: 19,
		WSMaxFrameBytes:        20,
		WSPingInterval:         21 * time.Second,
	}

	got := ConfigFromServer(cf)

	assert.Equal(t, serverlessoperator.Config{
		OperatorName:           "custom",
		LinkName:               LinkName,
		DefaultSlots:           11,
		DurableSlots:           12,
		LeaseTTL:               13 * time.Second,
		HeartbeatInterval:      14 * time.Second,
		RebalanceInterval:      15 * time.Second,
		ShedHysteresis:         0.3,
		DrainTimeout:           16 * time.Second,
		RoutingRefreshInterval: 17 * time.Second,
		HealthcheckTimeout:     18 * time.Second,
		HealthcheckConcurrency: 19,
		WSMaxFrameBytes:        20,
		WSPingInterval:         21 * time.Second,
		HealthPort:             0,
	}, got)

	// the engine serves health and metrics itself; the core's server stays off even when
	// the config file is empty
	assert.Equal(t, 0, ConfigFromServer(server.ServerlessOperatorConfigFile{}).HealthPort)
	assert.Equal(t, LinkName, ConfigFromServer(server.ServerlessOperatorConfigFile{}).LinkName)
}
