//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

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

func runsE2ETriggerEcho(t *testing.T, h *testharness.CLIHarness) string {
	t.Helper()

	inputFile := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	out := h.RunJSON("trigger", "manual", "--workflow", "e2e-echo", "--json", inputFile)

	var result struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal trigger output: %v\nOutput: %s", err, out)
	}
	if result.RunID == "" {
		t.Fatalf("expected runId in trigger response, got: %s", out)
	}
	return result.RunID
}

func runsE2EPollStatus(t *testing.T, h *testharness.CLIHarness, runID string) string {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	last := ""
	for time.Now().Before(deadline) {
		cmd := exec.Command(h.BinaryPath, "runs", "status", runID, "-o", "json", "--profile", h.Profile)
		out, err := cmd.Output()
		if err == nil {
			var status string
			if jsonErr := json.Unmarshal(out, &status); jsonErr == nil && status != "" {
				last = status
				if status == "COMPLETED" || status == "FAILED" || status == "CANCELLED" {
					return status
				}
			}
		}
		time.Sleep(time.Second)
	}
	return last
}

func TestRunsStatusJSON(t *testing.T) {
	h := testharness.New(t)
	runID := runsE2ETriggerEcho(t, h)

	status := runsE2EPollStatus(t, h, runID)
	if status == "" {
		t.Fatalf("run %s never returned a status within 30s", runID)
	}
	if status != "COMPLETED" {
		t.Errorf("expected run %s to reach COMPLETED within 30s, last status: %q", runID, status)
	}
}

func TestRunsTimingsJSON(t *testing.T) {
	h := testharness.New(t)
	runID := runsE2ETriggerEcho(t, h)

	if status := runsE2EPollStatus(t, h, runID); status != "COMPLETED" {
		t.Fatalf("expected run %s to reach COMPLETED before fetching timings, last status: %q", runID, status)
	}

	out := h.RunJSON("runs", "timings", runID)

	var timings interface{}
	if err := json.Unmarshal(out, &timings); err != nil {
		t.Fatalf("failed to unmarshal runs timings output: %v\nOutput: %s", err, out)
	}
	if timings == nil {
		t.Errorf("expected non-null timings response, got: %s", out)
	}
}

func TestRunsMetricsJSON(t *testing.T) {
	h := testharness.New(t)

	out := h.RunJSON("runs", "metrics")
	var metrics interface{}
	if err := json.Unmarshal(out, &metrics); err != nil {
		t.Fatalf("failed to unmarshal runs metrics output: %v\nOutput: %s", err, out)
	}
	if metrics == nil {
		t.Errorf("expected non-null metrics response, got: %s", out)
	}

	pointsOut := h.RunJSON("runs", "metrics", "--points")
	var points interface{}
	if err := json.Unmarshal(pointsOut, &points); err != nil {
		t.Fatalf("failed to unmarshal runs metrics --points output: %v\nOutput: %s", err, pointsOut)
	}
	if points == nil {
		t.Errorf("expected non-null point metrics response, got: %s", pointsOut)
	}
}

func TestRunsRestoreNonExistent(t *testing.T) {
	h := testharness.New(t)

	cmd := exec.Command(h.BinaryPath, "runs", "restore", uuid.NewString(), "-o", "json", "--profile", h.Profile)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected runs restore to fail for a nonexistent task, got success with output: %s", output)
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected a non-zero exit from runs restore, got: %v", err)
	}
	if len(output) == 0 {
		t.Error("expected an error message from runs restore, got empty output")
	}
}
