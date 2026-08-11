//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
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

func scheduledE2ECreateEcho(t *testing.T, h *testharness.CLIHarness, triggerAt time.Time) string {
	t.Helper()

	out := h.RunJSON("scheduled", "create",
		"--workflow", "e2e-echo",
		"--trigger-at", triggerAt.UTC().Format(time.RFC3339),
		"--input", "{}",
	)

	var created struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &created); err != nil {
		t.Fatalf("failed to unmarshal scheduled create output: %v\nOutput: %s", err, out)
	}
	if created.Metadata.ID == "" {
		t.Fatalf("expected metadata.id in scheduled create response, got: %s", out)
	}
	return created.Metadata.ID
}

func scheduledE2ECleanup(h *testharness.CLIHarness, ids ...string) {
	for _, id := range ids {
		_ = exec.Command(h.BinaryPath, "scheduled", "delete", id, "--yes", "-o", "json", "--profile", h.Profile).Run()
	}
}

func TestScheduledUpdateTriggerJSON(t *testing.T) {
	h := testharness.New(t)

	scheduledID := scheduledE2ECreateEcho(t, h, time.Now().Add(24*time.Hour))
	t.Cleanup(func() { scheduledE2ECleanup(h, scheduledID) })

	newTriggerAt := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	updateOut := h.RunJSON("scheduled", "update", scheduledID, "--trigger-at", newTriggerAt.Format(time.RFC3339))

	var updated struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
		TriggerAt time.Time `json:"triggerAt"`
	}
	if err := json.Unmarshal(updateOut, &updated); err != nil {
		t.Fatalf("failed to unmarshal scheduled update output: %v\nOutput: %s", err, updateOut)
	}
	if updated.Metadata.ID != scheduledID {
		t.Errorf("expected metadata.id = %q, got %q", scheduledID, updated.Metadata.ID)
	}
	if !updated.TriggerAt.Equal(newTriggerAt) {
		t.Errorf("expected triggerAt = %s, got %s", newTriggerAt.Format(time.RFC3339), updated.TriggerAt.Format(time.RFC3339))
	}

	triggerOut := h.RunJSON("scheduled", "trigger", scheduledID)

	var triggered struct {
		ExternalID string `json:"externalId"`
	}
	if err := json.Unmarshal(triggerOut, &triggered); err != nil {
		t.Fatalf("failed to unmarshal scheduled trigger output: %v\nOutput: %s", err, triggerOut)
	}
	if triggered.ExternalID == "" {
		t.Errorf("expected externalId in scheduled trigger response, got: %s", triggerOut)
	}
}

func TestScheduledBulkUpdateDeleteJSON(t *testing.T) {
	h := testharness.New(t)

	id1 := scheduledE2ECreateEcho(t, h, time.Now().Add(24*time.Hour))
	id2 := scheduledE2ECreateEcho(t, h, time.Now().Add(25*time.Hour))
	t.Cleanup(func() { scheduledE2ECleanup(h, id1, id2) })

	newTriggerAt := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	bulkOut := h.RunJSON("scheduled", "bulk-update", "--ids", id1+","+id2, "--trigger-at", newTriggerAt)

	var bulk struct {
		UpdatedIds []string                 `json:"updatedIds"`
		Errors     []map[string]interface{} `json:"errors"`
	}
	if err := json.Unmarshal(bulkOut, &bulk); err != nil {
		t.Fatalf("failed to unmarshal scheduled bulk-update output: %v\nOutput: %s", err, bulkOut)
	}
	if len(bulk.UpdatedIds) != 2 {
		t.Errorf("expected 2 updated ids, got %d: %s", len(bulk.UpdatedIds), bulkOut)
	}
	if len(bulk.Errors) != 0 {
		t.Errorf("expected no bulk-update errors, got: %s", bulkOut)
	}

	deleteOut := h.RunJSON("scheduled", "delete", id1, id2, "--yes")

	var deleted struct {
		DeletedIds []string `json:"deletedIds"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal scheduled bulk delete output: %v\nOutput: %s", err, deleteOut)
	}
	if len(deleted.DeletedIds) != 2 {
		t.Errorf("expected 2 deleted ids, got %d: %s", len(deleted.DeletedIds), deleteOut)
	}

	listOut := h.RunJSON("scheduled", "list", "--limit", "200")

	var list struct {
		Rows *[]struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(listOut, &list); err != nil {
		t.Fatalf("failed to unmarshal scheduled list output: %v\nOutput: %s", err, listOut)
	}
	if list.Rows != nil {
		for _, row := range *list.Rows {
			if row.Metadata.ID == id1 || row.Metadata.ID == id2 {
				t.Errorf("expected scheduled run %s to be deleted, but it is still listed", row.Metadata.ID)
			}
		}
	}
}
