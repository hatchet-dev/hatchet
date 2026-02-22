//go:build e2e_cli

package cli

import (
	"encoding/json"
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
