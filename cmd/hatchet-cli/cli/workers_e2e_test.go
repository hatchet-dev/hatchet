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

func TestWorkerPauseResumeJSON(t *testing.T) {
	h := testharness.New(t)

	out := h.RunJSON("worker", "list")
	var list struct {
		Rows []struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
			Name string `json:"name"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		t.Fatalf("failed to unmarshal worker list output: %v\nOutput: %s", err, out)
	}

	workerID := ""
	for _, row := range list.Rows {
		if row.Name == "e2e-worker" {
			workerID = row.Metadata.ID
			break
		}
	}
	if workerID == "" {
		t.Skip("worker 'e2e-worker' not found; skipping pause/resume test")
	}

	defer func() {
		cmd := exec.Command(h.BinaryPath, "worker", "resume", workerID, "-o", "json", "--profile", h.Profile)
		if err := cmd.Run(); err != nil {
			t.Logf("cleanup: failed to resume worker %s: %v", workerID, err)
		}
	}()

	type worker struct {
		IsPaused *bool `json:"isPaused"`
	}

	out = h.RunJSON("worker", "pause", workerID)
	var paused worker
	if err := json.Unmarshal(out, &paused); err != nil {
		t.Fatalf("failed to unmarshal worker pause output: %v\nOutput: %s", err, out)
	}
	if paused.IsPaused == nil || !*paused.IsPaused {
		t.Errorf("expected isPaused=true after pause; got: %s", out)
	}

	out = h.RunJSON("worker", "resume", workerID)
	var resumed worker
	if err := json.Unmarshal(out, &resumed); err != nil {
		t.Fatalf("failed to unmarshal worker resume output: %v\nOutput: %s", err, out)
	}
	if resumed.IsPaused != nil && *resumed.IsPaused {
		t.Errorf("expected isPaused=false after resume; got: %s", out)
	}
}
