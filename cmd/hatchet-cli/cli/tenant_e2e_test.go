//go:build e2e_cli

package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func e2eTryRunJSON(t *testing.T, h *testharness.CLIHarness, args ...string) ([]byte, error) {
	t.Helper()
	full := append(args, "-o", "json", "--profile", h.Profile)
	out, err := exec.Command(h.BinaryPath, full...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out, fmt.Errorf("%v: %s", err, exitErr.Stderr)
		}
		return out, err
	}
	return out, nil
}

func TestTenantGetJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("tenant", "get")

	var tenant struct {
		Name     string `json:"name"`
		Metadata struct {
			Id string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &tenant); err != nil {
		t.Fatalf("failed to unmarshal tenant get output: %v\nOutput: %s", err, out)
	}
	if tenant.Name == "" {
		t.Error("expected non-empty tenant name")
	}
	if tenant.Metadata.Id == "" {
		t.Error("expected non-empty tenant metadata id")
	}
}

func TestTenantUpdateJSON(t *testing.T) {
	h := testharness.New(t)

	getOut := h.RunJSON("tenant", "get")
	var current struct {
		AnalyticsOptOut *bool `json:"analyticsOptOut"`
	}
	if err := json.Unmarshal(getOut, &current); err != nil {
		t.Fatalf("failed to unmarshal tenant get output: %v\nOutput: %s", err, getOut)
	}
	original := current.AnalyticsOptOut != nil && *current.AnalyticsOptOut

	updateOut := h.RunJSON("tenant", "update", "--analytics-opt-out=true")
	var updated struct {
		Name            string `json:"name"`
		AnalyticsOptOut *bool  `json:"analyticsOptOut"`
	}
	if err := json.Unmarshal(updateOut, &updated); err != nil {
		t.Fatalf("failed to unmarshal tenant update output: %v\nOutput: %s", err, updateOut)
	}
	if updated.Name == "" {
		t.Error("expected non-empty tenant name in update response")
	}
	if updated.AnalyticsOptOut == nil || !*updated.AnalyticsOptOut {
		t.Errorf("expected analyticsOptOut=true after update, got: %s", updateOut)
	}

	h.RunJSON("tenant", "update", fmt.Sprintf("--analytics-opt-out=%t", original))
}

func TestTenantResourcePolicyJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("tenant", "resource-policy")

	var result map[string]json.RawMessage
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal resource-policy output: %v\nOutput: %s", err, out)
	}
	if _, ok := result["limits"]; !ok {
		t.Errorf("expected 'limits' field in resource policy, got: %s", out)
	}
}

func TestTenantQueueMetricsJSON(t *testing.T) {
	h := testharness.New(t)

	out := h.RunJSON("tenant", "queue-metrics")
	var metrics map[string]interface{}
	if err := json.Unmarshal(out, &metrics); err != nil {
		t.Fatalf("failed to unmarshal queue-metrics output: %v\nOutput: %s", err, out)
	}

	stepOut := h.RunJSON("tenant", "queue-metrics", "--step-runs")
	var stepMetrics map[string]interface{}
	if err := json.Unmarshal(stepOut, &stepMetrics); err != nil {
		t.Fatalf("failed to unmarshal step run queue-metrics output: %v\nOutput: %s", err, stepOut)
	}
}

func TestTenantTaskStatsJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("tenant", "task-stats")

	var stats interface{}
	if err := json.Unmarshal(out, &stats); err != nil {
		t.Fatalf("failed to unmarshal task-stats output: %v\nOutput: %s", err, out)
	}
}

func TestTenantPrometheusMetricsJSON(t *testing.T) {
	h := testharness.New(t)

	out, err := exec.Command(h.BinaryPath, "tenant", "prometheus-metrics", "--profile", h.Profile).Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		t.Skipf("prometheus-metrics unavailable (likely disabled server-side): %v\nStderr: %s", err, stderr)
	}
	if len(out) == 0 {
		t.Skip("prometheus-metrics returned empty output")
	}
}

func TestTenantMembersListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("tenant", "members", "list")

	var result struct {
		Rows []struct {
			Role     string `json:"role"`
			Metadata struct {
				Id string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal members list output: %v\nOutput: %s", err, out)
	}
	for i, row := range result.Rows {
		if row.Metadata.Id == "" {
			t.Errorf("row[%d]: expected non-empty member id", i)
		}
	}
}

func TestTenantInvitesCreateListDeleteJSON(t *testing.T) {
	h := testharness.New(t)

	email := fmt.Sprintf("e2e-invite-%d@example.com", time.Now().UnixNano())

	createOut, err := e2eTryRunJSON(t, h, "tenant", "invites", "create", "--email", email, "--role", "MEMBER")
	if err != nil {
		t.Skipf("invite creation not supported in this environment: %v", err)
	}

	var created struct {
		Email    string `json:"email"`
		Role     string `json:"role"`
		Metadata struct {
			Id string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("failed to unmarshal invite create output: %v\nOutput: %s", err, createOut)
	}
	if created.Email != email {
		t.Errorf("expected invite email %q, got %q", email, created.Email)
	}
	if created.Metadata.Id == "" {
		t.Fatalf("expected non-empty invite id, got: %s", createOut)
	}

	listOut := h.RunJSON("tenant", "invites", "list")
	var list struct {
		Rows []struct {
			Email    string `json:"email"`
			Metadata struct {
				Id string `json:"id"`
			} `json:"metadata"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(listOut, &list); err != nil {
		t.Fatalf("failed to unmarshal invites list output: %v\nOutput: %s", err, listOut)
	}
	found := false
	for _, row := range list.Rows {
		if row.Metadata.Id == created.Metadata.Id {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find invite %s in list, got: %s", created.Metadata.Id, listOut)
	}

	updateOut := h.RunJSON("tenant", "invites", "update", created.Metadata.Id, "--role", "ADMIN")
	var updated struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(updateOut, &updated); err != nil {
		t.Fatalf("failed to unmarshal invite update output: %v\nOutput: %s", err, updateOut)
	}
	if updated.Role != "ADMIN" {
		t.Errorf("expected role ADMIN after update, got %q", updated.Role)
	}

	deleteOut := h.RunJSON("tenant", "invites", "delete", "--yes", created.Metadata.Id)
	var deleted struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(deleteOut, &deleted); err != nil {
		t.Fatalf("failed to unmarshal invite delete output: %v\nOutput: %s", err, deleteOut)
	}
	if !deleted.Deleted {
		t.Errorf("expected deleted=true, got: %s", deleteOut)
	}

	afterOut := h.RunJSON("tenant", "invites", "list")
	if err := json.Unmarshal(afterOut, &list); err != nil {
		t.Fatalf("failed to unmarshal invites list output: %v\nOutput: %s", err, afterOut)
	}
	for _, row := range list.Rows {
		if row.Metadata.Id == created.Metadata.Id {
			t.Errorf("expected invite %s to be deleted, still in list", created.Metadata.Id)
		}
	}
}

func TestTenantFeatureFlagJSON(t *testing.T) {
	h := testharness.New(t)

	out, err := e2eTryRunJSON(t, h, "tenant", "feature-flag", "e2e-test-flag", "--enabled-if-no-posthog")
	if err != nil {
		t.Skipf("feature-flag evaluation not supported in this environment: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal feature-flag output: %v\nOutput: %s", err, out)
	}
}
