//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

func newClient(t *testing.T) *hatchet.Client {
	t.Helper()
	client, err := hatchet.NewClient()
	require.NoError(t, err)
	return client
}

func startWorker(t *testing.T, worker *hatchet.Worker) func() error {
	t.Helper()
	cleanup, err := worker.Start()
	require.NoError(t, err)
	time.Sleep(3 * time.Second)
	return cleanup
}

func bgCtx() context.Context {
	return context.Background()
}
