//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
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
