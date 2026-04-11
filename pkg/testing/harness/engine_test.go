//go:build e2e || integration

package harness

import (
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	RunTestWithEngine(m)
}

// Tests that the engine starts up and shuts down correctly without leaking
// goroutines.
func TestStartupShutdown(t *testing.T) {
	if err := WaitEngineReady(t.Context(), 120*time.Second); err != nil {
		t.Errorf("failed to bring up engine in time: %s", err)
	}
}
