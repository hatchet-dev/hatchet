//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestRunsListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("runs", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal runs list output: %v\nOutput: %s", err, out)
	}

	// rows may be empty on a fresh server, but the field must be present
	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}
}

func TestRunsGetJSON(t *testing.T) {
	runID := os.Getenv("HATCHET_TEST_RUN_ID")
	if runID == "" {
		t.Skip("HATCHET_TEST_RUN_ID not set; skipping runs get test")
	}

	h := testharness.New(t)
	out := h.RunJSON("runs", "get", runID)

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal runs get output: %v\nOutput: %s", err, out)
	}
	if result.Metadata.ID != runID {
		t.Errorf("expected metadata.id = %q, got %q", runID, result.Metadata.ID)
	}
}

func TestRunsTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("runs", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Runs", "runs") {
		t.Errorf("expected TUI to show 'Runs' header; got:\n%s", content)
	}
}
