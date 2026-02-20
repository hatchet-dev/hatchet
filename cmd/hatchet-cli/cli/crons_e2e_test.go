//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestCronListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("cron", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal cron list output: %v\nOutput: %s", err, out)
	}

	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}

	// If rows exist, validate expected fields
	for i, row := range result.Rows {
		if row["cron"] == nil && row["cronExpression"] == nil {
			t.Logf("row[%d]: cron expression field not found; available fields: %v", i, keys(row))
		}
	}
}

func TestCronCreateDisableDeleteJSON(t *testing.T) {
	workflowID := os.Getenv("HATCHET_TEST_WORKFLOW_ID")
	if workflowID == "" {
		t.Skip("HATCHET_TEST_WORKFLOW_ID not set; skipping cron create/disable/delete test")
	}

	h := testharness.New(t)

	// Create a cron job
	createOut := h.RunJSON("cron", "create",
		"--workflow", workflowID,
		"--cron", "0 * * * *",
		"--name", "e2e-test-cron",
	)

	var created struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal cron create output: %v\nOutput: %s", err, createOut)
	}
	if created.Metadata.ID == "" {
		t.Fatal("expected metadata.id in cron create response")
	}

	cronID := created.Metadata.ID
	t.Logf("Created cron job: %s", cronID)

	// Disable the cron job
	disableOut := h.RunJSON("cron", "disable", "--yes", cronID)

	var disabled struct {
		Enabled bool   `json:"enabled"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(disableOut, &disabled); err != nil {
		t.Fatalf("failed to unmarshal cron disable output: %v\nOutput: %s", err, disableOut)
	}
	if disabled.Enabled {
		t.Errorf("expected enabled=false after disable, got: %s", disableOut)
	}

	// Delete the cron job
	deleteOut := h.RunJSON("cron", "delete", "--yes", cronID)

	var deleted struct {
		Deleted bool   `json:"deleted"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal cron delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}
	if deleted.ID != cronID {
		t.Errorf("expected deleted id %q, got %q", cronID, deleted.ID)
	}

	t.Logf("Deleted cron job: %s", cronID)
}

func TestCronTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("cron", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Cron Jobs", "Cron", "Expression") {
		t.Errorf("expected TUI to show 'Cron Jobs' header; got:\n%s", content)
	}
}

// keys returns the map keys as a slice for logging purposes.
func keys(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
