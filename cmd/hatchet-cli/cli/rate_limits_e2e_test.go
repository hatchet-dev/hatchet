//go:build e2e_cli

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestRateLimitsListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("rate-limits", "list")

	var result struct {
		Rows []struct {
			Key        string `json:"key"`
			Value      int    `json:"value"`
			LimitValue int    `json:"limitValue"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal rate-limits list output: %v\nOutput: %s", err, out)
	}

	// rows may be empty, but the field must exist
	if result.Rows == nil {
		t.Errorf("expected 'rows' array in response, got nil")
	}

	// If there are rows, validate they have the expected fields
	for i, row := range result.Rows {
		if row.Key == "" {
			t.Errorf("row[%d]: expected non-empty 'key' field", i)
		}
	}
}

func TestRateLimitsTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("rate-limits", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Rate Limits", "rate-limits", "rate_limits") {
		t.Errorf("expected TUI to show 'Rate Limits' header; got:\n%s", content)
	}
}
