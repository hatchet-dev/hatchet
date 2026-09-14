//go:build e2e_cli

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func findEchoWorkflowID(t *testing.T, h *testharness.CLIHarness) string {
	t.Helper()
	out := h.RunJSON("workflows", "list")

	var result struct {
		Rows []struct {
			Name     string `json:"name"`
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal workflows list output: %v\nOutput: %s", err, out)
	}
	for _, row := range result.Rows {
		if row.Name == "e2e-echo" {
			return row.Metadata.ID
		}
	}
	t.Skip("workflow 'e2e-echo' not registered; skipping filters test")
	return ""
}

func listFilterIDsByScope(t *testing.T, h *testharness.CLIHarness, scope string) []string {
	t.Helper()
	out := h.RunJSON("filters", "list", "--scopes", scope)

	var result struct {
		Rows []struct {
			Metadata struct {
				ID string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal filters list output: %v\nOutput: %s", err, out)
	}
	ids := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		ids = append(ids, row.Metadata.ID)
	}
	return ids
}

func TestFiltersCreateGetListUpdateDeleteJSON(t *testing.T) {
	h := testharness.New(t)
	workflowID := findEchoWorkflowID(t, h)

	scope := "e2e-scope-filters-crud"
	expression := `input.message == "hi"`

	createOut := h.RunJSON("filters", "create",
		"--workflow-id", workflowID,
		"--expression", expression,
		"--scope", scope,
		"--payload", "{}",
	)

	var created struct {
		Expression string `json:"expression"`
		Scope      string `json:"scope"`
		WorkflowID string `json:"workflowId"`
		Metadata   struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal filters create output: %v\nOutput: %s", err, createOut)
	}
	if created.Metadata.ID == "" {
		t.Fatalf("expected non-empty filter id in create response\nOutput: %s", createOut)
	}
	if created.Expression != expression {
		t.Errorf("expected expression %q, got %q", expression, created.Expression)
	}
	if created.Scope != scope {
		t.Errorf("expected scope %q, got %q", scope, created.Scope)
	}
	if created.WorkflowID != workflowID {
		t.Errorf("expected workflowId %q, got %q", workflowID, created.WorkflowID)
	}

	filterID := created.Metadata.ID

	getOut := h.RunJSON("filters", "get", filterID)

	var got struct {
		Expression string `json:"expression"`
		Metadata   struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(getOut, &got); err != nil {
		t.Fatalf("failed to unmarshal filters get output: %v\nOutput: %s", err, getOut)
	}
	if got.Metadata.ID != filterID {
		t.Errorf("expected metadata.id = %q, got %q", filterID, got.Metadata.ID)
	}
	if got.Expression != expression {
		t.Errorf("expected expression %q, got %q", expression, got.Expression)
	}

	ids := listFilterIDsByScope(t, h, scope)
	foundInList := false
	for _, id := range ids {
		if id == filterID {
			foundInList = true
		}
	}
	if !foundInList {
		t.Errorf("expected filters list with scope %q to contain filter %q, got ids %v", scope, filterID, ids)
	}

	updatedExpression := `input.message == "bye"`
	updateOut := h.RunJSON("filters", "update", filterID, "--expression", updatedExpression)

	var updated struct {
		Expression string `json:"expression"`
		Metadata   struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(updateOut, &updated); err != nil {
		t.Fatalf("failed to unmarshal filters update output: %v\nOutput: %s", err, updateOut)
	}
	if updated.Metadata.ID != filterID {
		t.Errorf("expected metadata.id = %q, got %q", filterID, updated.Metadata.ID)
	}
	if updated.Expression != updatedExpression {
		t.Errorf("expected updated expression %q, got %q", updatedExpression, updated.Expression)
	}

	deleteOut := h.RunJSON("filters", "delete", "--yes", filterID)

	var deleted struct {
		Deleted bool   `json:"deleted"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal filters delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}
	if deleted.ID != filterID {
		t.Errorf("expected deleted id %q, got %q", filterID, deleted.ID)
	}

	for _, id := range listFilterIDsByScope(t, h, scope) {
		if id == filterID {
			t.Errorf("expected filters list to no longer contain deleted filter %q", filterID)
		}
	}
}

func TestFiltersTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("filters", "list")
	content := tui.CaptureAfter(4 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Filters", "filters") {
		t.Errorf("expected TUI to show 'Filters' header; got:\n%s", content)
	}
}
