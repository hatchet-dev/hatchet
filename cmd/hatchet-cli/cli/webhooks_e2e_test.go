//go:build e2e_cli

package cli

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

func TestWebhooksListJSON(t *testing.T) {
	h := testharness.New(t)
	out := h.RunJSON("webhooks", "list")

	var result map[string]interface{}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal webhooks list output: %v\nOutput: %s", err, out)
	}

	if result == nil {
		t.Error("expected non-nil JSON response from webhooks list")
	}
}

func TestWebhooksTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("webhooks", "list")
	content := tui.CaptureAfter(3 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Webhooks", "webhooks") {
		t.Errorf("expected TUI to show 'Webhooks' header; got:\n%s", content)
	}
}

func TestWebhooksCreateGetUpdateDeleteJSON(t *testing.T) {
	h := testharness.New(t)
	name := fmt.Sprintf("e2e-wh-%d", time.Now().UnixNano())

	type webhook struct {
		Name               string `json:"name"`
		EventKeyExpression string `json:"eventKeyExpression"`
	}

	out := h.RunJSON("webhooks", "create",
		"--name", name,
		"--source", "GENERIC",
		"--event-key-expression", "input.event",
		"--auth-type", "basic",
		"--username", "e2e-user",
		"--password", "e2e-pass",
	)
	var created webhook
	if err := json.Unmarshal(out, &created); err != nil {
		t.Fatalf("failed to unmarshal webhooks create output: %v\nOutput: %s", err, out)
	}
	if created.Name != name {
		t.Fatalf("expected created webhook name %q, got %q", name, created.Name)
	}

	out = h.RunJSON("webhooks", "get", name)
	var got webhook
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to unmarshal webhooks get output: %v\nOutput: %s", err, out)
	}
	if got.Name != name {
		t.Errorf("expected webhook name %q, got %q", name, got.Name)
	}

	out = h.RunJSON("webhooks", "update", name, "--event-key-expression", "input.event_type")
	var updated webhook
	if err := json.Unmarshal(out, &updated); err != nil {
		t.Fatalf("failed to unmarshal webhooks update output: %v\nOutput: %s", err, out)
	}
	if updated.EventKeyExpression != "input.event_type" {
		t.Errorf("expected updated eventKeyExpression %q, got %q", "input.event_type", updated.EventKeyExpression)
	}

	out = h.RunJSON("webhooks", "delete", name, "--yes")
	var deleted struct {
		Deleted bool   `json:"deleted"`
		Name    string `json:"name"`
	}
	if err := json.Unmarshal(out, &deleted); err != nil {
		t.Fatalf("failed to unmarshal webhooks delete output: %v\nOutput: %s", err, out)
	}
	if !deleted.Deleted || deleted.Name != name {
		t.Errorf("expected deleted=true and name=%q, got: %s", name, out)
	}

	out = h.RunJSON("webhooks", "list")
	var list struct {
		Rows []webhook `json:"rows"`
	}
	if err := json.Unmarshal(out, &list); err != nil {
		t.Fatalf("failed to unmarshal webhooks list output: %v\nOutput: %s", err, out)
	}
	for _, row := range list.Rows {
		if row.Name == name {
			t.Errorf("expected webhook %q to be deleted, but it is still listed", name)
		}
	}
}
