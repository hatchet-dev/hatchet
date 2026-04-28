//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	defaultTimeout = 5 * time.Minute
	pollInterval   = 200 * time.Millisecond
	maxPolls       = 150
)

var (
	sharedClient  *hatchet.Client
	sharedCleanup func() error
)

func TestMain(m *testing.M) {
	client, err := hatchet.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create hatchet client: %v\n", err)
		os.Exit(1)
	}
	sharedClient = client

	worker, cleanup, err := startTestWorker(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test worker: %v\n", err)
		os.Exit(1)
	}
	_ = worker
	sharedCleanup = cleanup

	code := m.Run()

	if sharedCleanup != nil {
		_ = sharedCleanup()
	}

	os.Exit(code)
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	t.Cleanup(cancel)
	return ctx
}

func uniqueID() string {
	return uuid.New().String()
}

// pollUntil polls fn every pollInterval until it returns true or maxPolls is reached.
func pollUntil(t *testing.T, ctx context.Context, fn func() (bool, error)) {
	t.Helper()
	for i := 0; i < maxPolls; i++ {
		done, err := fn()
		if err != nil {
			t.Logf("poll error (attempt %d): %v", i, err)
		}
		if done {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while polling: %v", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
	t.Fatalf("polling timed out after %d attempts", maxPolls)
}

// pollUntilRunStatus polls run details until any task reaches the given status.
func pollUntilRunStatus(
	t *testing.T,
	ctx context.Context,
	client *hatchet.Client,
	runID string,
	targetStatus string,
) {
	t.Helper()
	pollUntil(t, ctx, func() (bool, error) {
		details, err := client.Runs().GetDetails(ctx, uuid.MustParse(runID))
		if err != nil {
			return false, err
		}
		for _, task := range details.TaskRuns {
			if string(task.Status) == targetStatus {
				return true, nil
			}
		}
		return false, nil
	})
}

// pollUntilEvicted polls run details until any task has IsEvicted=true.
func pollUntilEvicted(
	t *testing.T,
	ctx context.Context,
	client *hatchet.Client,
	runID string,
) {
	t.Helper()
	pollUntil(t, ctx, func() (bool, error) {
		details, err := client.Runs().GetDetails(ctx, uuid.MustParse(runID))
		if err != nil {
			return false, err
		}
		for _, task := range details.TaskRuns {
			if task.IsEvicted {
				return true, nil
			}
		}
		return false, nil
	})
}

// requireDurableEviction skips the test if the engine does not support durable eviction.
func requireDurableEviction(t *testing.T) {
	t.Helper()
	version, err := sharedClient.GetEngineVersion(context.Background())
	if err != nil {
		t.Skipf("could not get engine version: %v", err)
	}
	if version == "" {
		t.Skip("engine version is empty, skipping durable eviction test")
	}

	supported, err := hatchet.SupportsDurableEviction(version)
	if err != nil {
		t.Skipf("could not check durable eviction support: %v", err)
	}
	if !supported {
		t.Skipf("engine %s does not support durable eviction", version)
	}
}

func resultMap(t *testing.T, result *hatchet.WorkflowResult, taskName string) map[string]any {
	t.Helper()
	var m map[string]any
	err := result.TaskOutput(taskName).Into(&m)
	require.NoError(t, err, "failed to parse task output for %q", taskName)
	return m
}
