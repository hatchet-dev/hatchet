//go:build e2e_cli

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/testharness"
)

const echoEventKey = "e2e:echo"

func retryUntil(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Second)
	}
}

func pushEchoEvent(t *testing.T, h *testharness.CLIHarness, message string) {
	t.Helper()
	out := h.RunJSON("events", "push", echoEventKey, "--data", `{"message":"`+message+`"}`)

	var event struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(out, &event); err != nil {
		t.Fatalf("failed to unmarshal events push output: %v\nOutput: %s", err, out)
	}
	if event.Key != echoEventKey {
		t.Fatalf("expected event key %q, got %q", echoEventKey, event.Key)
	}
}

func TestEventsPushJSON(t *testing.T) {
	h := testharness.New(t)
	pushEchoEvent(t, h, "hi")
}

func TestEventsBulkPushJSON(t *testing.T) {
	h := testharness.New(t)

	events := []map[string]interface{}{
		{"key": echoEventKey, "data": map[string]interface{}{"message": "e2e-evt-bulk-1"}},
		{"key": echoEventKey, "data": map[string]interface{}{"message": "e2e-evt-bulk-2"}},
	}
	raw, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("failed to marshal events: %v", err)
	}
	path := filepath.Join(t.TempDir(), "events.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("failed to write events file: %v", err)
	}

	out := h.RunJSON("events", "bulk-push", "--file", path)

	var result struct {
		Events []struct {
			Key string `json:"key"`
		} `json:"events"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("failed to unmarshal events bulk-push output: %v\nOutput: %s", err, out)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events in response, got %d\nOutput: %s", len(result.Events), out)
	}
	for i, event := range result.Events {
		if event.Key != echoEventKey {
			t.Errorf("event[%d]: expected key %q, got %q", i, echoEventKey, event.Key)
		}
	}
}

func TestEventsListJSON(t *testing.T) {
	h := testharness.New(t)
	pushEchoEvent(t, h, "e2e-evt-list")

	var lastOut []byte
	found := retryUntil(t, 10*time.Second, func() bool {
		lastOut = h.RunJSON("events", "list", "--keys", echoEventKey, "--limit", "10")

		var result struct {
			Rows []struct {
				Key string `json:"key"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(lastOut, &result); err != nil {
			t.Fatalf("failed to unmarshal events list output: %v\nOutput: %s", err, lastOut)
		}
		if result.Rows == nil {
			return false
		}
		for _, row := range result.Rows {
			if row.Key == echoEventKey {
				return true
			}
		}
		return false
	})
	if !found {
		t.Fatalf("expected events list to contain a row with key %q\nLast output: %s", echoEventKey, lastOut)
	}
}

func TestEventsGetJSON(t *testing.T) {
	h := testharness.New(t)
	pushEchoEvent(t, h, "e2e-evt-get")

	var eventID string
	found := retryUntil(t, 10*time.Second, func() bool {
		out := h.RunJSON("events", "list", "--keys", echoEventKey, "--limit", "10")

		var result struct {
			Rows []struct {
				Key      string `json:"key"`
				Metadata struct {
					ID string `json:"id"`
				} `json:"metadata"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(out, &result); err != nil {
			t.Fatalf("failed to unmarshal events list output: %v\nOutput: %s", err, out)
		}
		for _, row := range result.Rows {
			if row.Key == echoEventKey && row.Metadata.ID != "" {
				eventID = row.Metadata.ID
				return true
			}
		}
		return false
	})
	if !found {
		t.Fatalf("could not find a pushed event with key %q to get", echoEventKey)
	}

	out := h.RunJSON("events", "get", eventID)

	var event struct {
		Metadata struct {
			ID string `json:"id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &event); err != nil {
		t.Fatalf("failed to unmarshal events get output: %v\nOutput: %s", err, out)
	}
	if event.Metadata.ID != eventID {
		t.Errorf("expected metadata.id = %q, got %q", eventID, event.Metadata.ID)
	}
}

func TestEventsKeysJSON(t *testing.T) {
	h := testharness.New(t)
	pushEchoEvent(t, h, "e2e-evt-keys")

	var lastOut []byte
	found := retryUntil(t, 10*time.Second, func() bool {
		lastOut = h.RunJSON("events", "keys")

		var result struct {
			Rows []string `json:"rows"`
		}
		if err := json.Unmarshal(lastOut, &result); err != nil {
			t.Fatalf("failed to unmarshal events keys output: %v\nOutput: %s", err, lastOut)
		}
		for _, key := range result.Rows {
			if key == echoEventKey {
				return true
			}
		}
		return false
	})
	if !found {
		t.Fatalf("expected events keys to contain %q\nLast output: %s", echoEventKey, lastOut)
	}
}

func TestEventsTUI(t *testing.T) {
	h := testharness.New(t)
	tui := testharness.NewTUI(t, h)
	t.Cleanup(tui.Stop)

	tui.Start("events", "list")
	content := tui.CaptureAfter(4 * time.Second)

	if content == "" {
		t.Fatal("TUI output was empty")
	}
	if !containsAny(content, "Events", "events") {
		t.Errorf("expected TUI to show 'Events' header; got:\n%s", content)
	}
}
