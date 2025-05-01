//go:build e2e

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
	time.Sleep(5 * time.Second)
}
