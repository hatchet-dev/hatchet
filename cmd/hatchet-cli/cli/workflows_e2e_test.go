//go:build e2e_cli

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestWorkflowsListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("workflows", "list")

	var result struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows list output: %v\nOutput: %s", err, out)
	}

	// rows may be empty, but the field must exist
	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}
}

func TestWorkflowsGetJSON(t *testing.T) {
	workflowID := os.Getenv("HATCHET_TEST_WORKFLOW_ID")
	if workflowID == "" {
		t.Skip("HATCHET_TEST_WORKFLOW_ID not set; skipping workflows get test")
	}

	h := testharness.New(t)
	out := h.RunJSON("workflows", "get", workflowID)

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows get output: %v\nOutput: %s", err, out)
	}
	if result.Metadata.ID != workflowID {
		t.Errorf("expected metadata.id = %q, got %q", workflowID, result.Metadata.ID)
	}
}

func TestWorkflowsTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("workflows", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Workflows", "workflows") {
		t.Errorf("expected TUI to show 'Workflows' header; got:\n%s", content)
	}
}

// containsAny reports whether s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func findWorkflowID(t *testing.T, h *testharness.CLIHarness, name string) string {
	t.Helper()

	out := h.RunJSON("workflows", "list", "--search", name)
	var result struct {
		Rows []struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
			Name string `json:"name"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows list output: %v\nOutput: %s", err, out)
	}
	for _, row := range result.Rows {
		if row.Name == name {
			return row.Metadata.ID
		}
	}
	return ""
}

func requireEchoWorkflowID(t *testing.T, h *testharness.CLIHarness) string {
	t.Helper()

	id := findWorkflowID(t, h, "e2e-echo")
	if id == "" {
		t.Skip("workflow 'e2e-echo' not found; skipping")
	}
	return id
}

func TestWorkflowsPauseUnpauseJSON(t *testing.T) {
	h := testharness.New(t)
	workflowID := requireEchoWorkflowID(t, h)

	defer func() {
		cmd := exec.Command(h.BinaryPath, "workflows", "unpause", workflowID, "-o", "json", "--profile", h.Profile)
		if err := cmd.Run(); err != nil {
			t.Logf("cleanup: failed to unpause workflow %s: %v", workflowID, err)
		}
	}()

	type workflow struct {
		IsPaused *bool `json:"isPaused"`
	}

	out := h.RunJSON("workflows", "pause", workflowID)
	var paused workflow
	if err := json.Unmarshal(out, &paused); err != nil {
		t.Fatalf("failed to unmarshal workflows pause output: %v\nOutput: %s", err, out)
	}
	if paused.IsPaused == nil || !*paused.IsPaused {
		t.Errorf("expected isPaused=true after pause; got: %s", out)
	}

	out = h.RunJSON("workflows", "unpause", workflowID)
	var unpaused workflow
	if err := json.Unmarshal(out, &unpaused); err != nil {
		t.Fatalf("failed to unmarshal workflows unpause output: %v\nOutput: %s", err, out)
	}
	if unpaused.IsPaused != nil && *unpaused.IsPaused {
		t.Errorf("expected isPaused=false after unpause; got: %s", out)
	}
}

func TestWorkflowsMetricsJSON(t *testing.T) {
	h := testharness.New(t)
	workflowID := requireEchoWorkflowID(t, h)

	out := h.RunJSON("workflows", "metrics", workflowID)

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows metrics output: %v\nOutput: %s", err, out)
	}
	if result == nil {
		t.Error("expected non-nil JSON response from workflows metrics")
	}
}

func TestWorkflowsWorkersCountJSON(t *testing.T) {
	h := testharness.New(t)
	workflowID := requireEchoWorkflowID(t, h)

	out := h.RunJSON("workflows", "workers-count", workflowID)

	var result struct {
		FreeSlotCount *int `json:"freeSlotCount"`
		MaxSlotCount  *int `json:"maxSlotCount"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workers-count output: %v\nOutput: %s", err, out)
	}
	if result.FreeSlotCount == nil || result.MaxSlotCount == nil {
		t.Errorf("expected freeSlotCount and maxSlotCount in response; got: %s", out)
	} else if *result.FreeSlotCount < 0 || *result.MaxSlotCount < 0 {
		t.Errorf("expected non-negative slot counts; got free=%d max=%d", *result.FreeSlotCount, *result.MaxSlotCount)
	}
}

func TestWorkflowsVersionJSON(t *testing.T) {
	h := testharness.New(t)
	workflowID := requireEchoWorkflowID(t, h)

	out := h.RunJSON("workflows", "version", workflowID)

	var result struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows version output: %v\nOutput: %s", err, out)
	}
	if result.Metadata.ID == "" {
		t.Errorf("expected workflow version metadata.id in response; got: %s", out)
	}
}

func TestWorkflowsDeleteJSON(t *testing.T) {
	h := testharness.New(t)

	nonexistentID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	cmd := exec.Command(h.BinaryPath, "workflows", "delete", nonexistentID, "--yes", "-o", "json", "--profile", h.Profile)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit deleting nonexistent workflow; output: %s", out)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		t.Error("expected an error message when deleting a nonexistent workflow")
	}
}
