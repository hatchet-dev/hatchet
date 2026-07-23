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

func TestRateLimitsCreateGetDeleteJSON(t *testing.T) {
	h := testharness.New(t)

	key := "e2e-test-rate-limit"

	// Create (upsert) a rate limit over the gRPC admin API
	createOut := h.RunJSON("rate-limits", "create",
		"--key", key,
		"--limit", "100",
		"--duration", "minute",
	)

	var created struct {
		Key      string `json:"key"`
		Limit    int    `json:"limit"`
		Duration string `json:"duration"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal rate-limits create output: %v\nOutput: %s", err, createOut)
	}
	if created.Key != key {
		t.Errorf("expected key %q, got %q", key, created.Key)
	}
	if created.Limit != 100 {
		t.Errorf("expected limit 100, got %d", created.Limit)
	}
	if created.Duration != "minute" {
		t.Errorf("expected duration minute, got %q", created.Duration)
	}

	// Get the rate limit by key
	getOut := h.RunJSON("rate-limits", "get", key)

	var got struct {
		Key        string `json:"key"`
		LimitValue int    `json:"limitValue"`
	}
	if err := json.Unmarshal(getOut, &got); err != nil {
		t.Fatalf("failed to unmarshal rate-limits get output: %v\nOutput: %s", err, getOut)
	}
	if got.Key != key {
		t.Errorf("expected key %q, got %q", key, got.Key)
	}

	// Delete the rate limit
	deleteOut := h.RunJSON("rate-limits", "delete", "--yes", key)

	var deleted struct {
		Deleted bool   `json:"deleted"`
		Key     string `json:"key"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal rate-limits delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}
	if deleted.Key != key {
		t.Errorf("expected deleted key %q, got %q", key, deleted.Key)
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
