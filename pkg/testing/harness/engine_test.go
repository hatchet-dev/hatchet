package harness

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

func TestMain(m *testing.M) {
	RunTestWithEngine(m)
}

// Tests that the engine starts up and shuts down correctly without leaking
// goroutines.
func TestStartupShutdown(t *testing.T) {
	if err := WaitEngineReady(t.Context(), 120*time.Second); err != nil {
		t.Fatalf("failed to bring up engine in time: %s", err)
	}

	// /ready answers as soon as the queue and database are healthy, which is
	// before the controllers and the gRPC server have started. The gRPC
	// listener is bound by the last start step, and GetVersion is an
	// authenticated RPC that only this engine's dispatcher answers, so a
	// successful call shows that the engine finished starting rather than
	// that something is listening on the port.
	if err := waitEngineGRPC(t.Context(), 60*time.Second); err != nil {
		t.Fatalf("engine did not finish starting in time: %s", err)
	}
}

func waitEngineGRPC(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// the client reads HATCHET_CLIENT_TOKEN, set by the harness; the token
	// carries the engine's gRPC broadcast address
	c, err := client.New()
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer c.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		_, err := c.Dispatcher().GetVersion(ctx)
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("GetVersion: %w", err)
		case <-ticker.C:
		}
	}
}
