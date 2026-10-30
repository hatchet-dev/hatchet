//go:build e2e

// Package e2e exercises the serverless operator core end to end against an in-process engine
// started by the test harness. The engine runs the in-engine operator (over the in-process
// host) as one process whose id is the dispatcher id; the tests start additional
// out-of-process instances (over the gRPC host) inside the same test binary, so every
// instance shares the lease table and each scenario can be run against either host by
// pinning its tenant's lease to the process it wants (see pinLease in env_test.go).
package e2e

import (
	"os"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
)

func TestMain(m *testing.M) {
	os.Setenv("SERVER_GRPC_OPERATORS_ENABLED", "true")
	os.Setenv("SERVER_SERVERLESS_OPERATOR_ENABLED", "true")
	os.Setenv("SERVER_SERVERLESS_OPERATOR_INSECURE_DESTINATIONS", "true")
	os.Setenv("SERVER_SERVERLESS_OPERATOR_ALLOW_EMPTY_INFRA_CIDRS", "true")
	os.Setenv("SERVER_SERVERLESS_OPERATOR_LEASE_TTL", leaseTTL.String())
	os.Setenv("SERVER_SERVERLESS_OPERATOR_HEARTBEAT_INTERVAL", tick.String())
	os.Setenv("SERVER_SERVERLESS_OPERATOR_REBALANCE_INTERVAL", tick.String())
	os.Setenv("SERVER_SERVERLESS_OPERATOR_ROUTING_REFRESH_INTERVAL", tick.String())
	os.Setenv("SERVER_SERVERLESS_OPERATOR_DRAIN_TIMEOUT", drainTimeout.String())

	harness.RunTestWithEngine(m)
}
