//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestWorkerListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("worker", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal worker list output: %v\nOutput: %s", err, out)
	}

	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}
}

func TestWorkerGetJSON(t *testing.T) {
	workerID := os.Getenv("HATCHET_TEST_WORKER_ID")
	if workerID == "" {
		t.Skip("HATCHET_TEST_WORKER_ID not set; skipping worker get test")
	}

	h := testharness.New(t)
	out := h.RunJSON("worker", "get", workerID)

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal worker get output: %v\nOutput: %s", err, out)
	}
	if result.Metadata.ID != workerID {
		t.Errorf("expected metadata.id = %q, got %q", workerID, result.Metadata.ID)
	}
}

func TestWorkersTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("worker", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Workers", "workers") {
		t.Errorf("expected TUI to show 'Workers' header; got:\n%s", content)
	}
}
