package harness

import (
	"context"
	"fmt"
	"net"
	"os"
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
		t.Fatalf("failed to bring up engine in time: %s", err)
	}

	// /ready answers as soon as the queue and database are healthy, which is
	// before the controllers and the gRPC server have started. The gRPC
	// listener is bound by the last start step, so a successful dial shows
	// that the engine finished starting rather than only becoming ready.
	if err := waitEngineGRPC(t.Context(), 60*time.Second); err != nil {
		t.Fatalf("engine did not finish starting in time: %s", err)
	}
}

func waitEngineGRPC(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	port := os.Getenv("SERVER_GRPC_PORT")
	if port == "" {
		return fmt.Errorf("SERVER_GRPC_PORT not set")
	}

	addr := net.JoinHostPort("localhost", port)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("dial %s: %w", addr, err)
		case <-ticker.C:
		}
	}
}
