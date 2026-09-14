//go:build e2e_cli

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestLogsListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("logs", "list", "--limit", "10")

	var result struct {
		Rows *[]struct {
			Message   string    `json:"message"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal logs list output: %v\nOutput: %s", err, out)
	}
	if result.Rows == nil {
		t.Errorf("expected 'rows' field in response, got: %s", out)
	}
}

func TestLogsListFiltersJSON(t *testing.T) {
	h := testharness.New(t)
	since := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	out := h.RunJSON("logs", "list", "--limit", "5", "--since", since, "--order", "desc")

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal logs list output: %v\nOutput: %s", err, out)
	}
	if _, ok := result["rows"]; !ok {
		t.Errorf("expected 'rows' field in response, got: %s", out)
	}
}

func TestLogsMetricsJSON(t *testing.T) {
	h := testharness.New(t)
	since := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	out := h.RunJSON("logs", "metrics", "--since", since)

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal logs metrics output: %v\nOutput: %s", err, out)
	}
}
