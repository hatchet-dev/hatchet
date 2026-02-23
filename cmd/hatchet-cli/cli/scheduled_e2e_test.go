//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestScheduledListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("scheduled", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal scheduled list output: %v\nOutput: %s", err, out)
	}

	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}
}

func TestScheduledCreateDeleteJSON(t *testing.T) {
	workflowID := os.Getenv("HATCHET_TEST_WORKFLOW_ID")
	if workflowID == "" {
		t.Skip("HATCHET_TEST_WORKFLOW_ID not set; skipping scheduled create/delete test")
	}

	// Use a trigger time 1 hour in the future
	triggerAt := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)

	h := testharness.New(t)

	// Create a scheduled run
	createOut := h.RunJSON("scheduled", "create",
		"--workflow", workflowID,
		"--trigger-at", triggerAt,
		"--input", "{}",
	)

	var created struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal scheduled create output: %v\nOutput: %s", err, createOut)
	}
	if created.Metadata.ID == "" {
		t.Fatal("expected metadata.id in scheduled create response")
	}

	scheduledID := created.Metadata.ID
	t.Logf("Created scheduled run: %s", scheduledID)

	// Delete the scheduled run
	deleteOut := h.RunJSON("scheduled", "delete", "--yes", scheduledID)

	var deleted struct {
		Deleted bool   `json:"deleted"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal scheduled delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}
	if deleted.ID != scheduledID {
		t.Errorf("expected deleted id %q, got %q", scheduledID, deleted.ID)
	}

	t.Logf("Deleted scheduled run: %s", scheduledID)
}

func TestScheduledGetJSON(t *testing.T) {
	scheduledID := os.Getenv("HATCHET_TEST_SCHEDULED_ID")
	if scheduledID == "" {
		t.Skip("HATCHET_TEST_SCHEDULED_ID not set; skipping scheduled get test")
	}

	h := testharness.New(t)
	out := h.RunJSON("scheduled", "get", scheduledID)

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal scheduled get output: %v\nOutput: %s", err, out)
	}
	if result.Metadata.ID != scheduledID {
		t.Errorf("expected metadata.id = %q, got %q", scheduledID, result.Metadata.ID)
	}
}

func TestScheduledTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("scheduled", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Scheduled Runs", "Scheduled", "Trigger") {
		t.Errorf("expected TUI to show 'Scheduled Runs' header; got:\n%s", content)
	}
}
