//go:build load

package harness

import (
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// This runs before all tests
	t := &testing.T{}
	postRun := StartEngine(t)

	// Run all tests in this package
	code := m.Run()

	if code != 0 {
		log.Printf("TestMain: code %d", code)
		os.Exit(code)
	}

	postRun()

	// determine if t is failed
	if t.Failed() {
		log.Printf("TestMain: test failed")
		os.Exit(1)
	}
}

// Tests that the engine starts up and shuts down correctly without leaking
// goroutines.
func TestStartupShutdown(t *testing.T) {
	time.Sleep(5 * time.Second)
}
