package posthog

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
)

// fakeClient embeds posthog.Client to satisfy the methods flushCount never
// calls.
type fakeClient struct {
	posthog.Client
	messages []posthog.Message
}

func (f *fakeClient) Enqueue(m posthog.Message) error {
	f.messages = append(f.messages, m)
	return nil
}

func newTestAnalytics(fake *fakeClient) *PosthogAnalytics {
	l := zerolog.Nop()
	var client posthog.Client = fake
	return &PosthogAnalytics{client: &client, l: &l}
}

func captureFrom(t *testing.T, fake *fakeClient) posthog.Capture {
	t.Helper()
	if len(fake.messages) != 1 {
		t.Fatalf("expected 1 enqueued message, got %d", len(fake.messages))
	}
	capture, ok := fake.messages[0].(posthog.Capture)
	if !ok {
		t.Fatalf("expected a posthog.Capture, got %T", fake.messages[0])
	}
	return capture
}

func TestFlushCount_GeneratedEventTimesWin(t *testing.T) {
	fake := &fakeClient{}
	p := newTestAnalytics(fake)

	first := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	last := first.Add(time.Minute)

	p.flushCount(analytics.Worker, analytics.Register, uuid.New(), nil, 5, first, last, analytics.Properties{
		"aggregate_first_event_at": "spoofed",
		"aggregate_last_event_at":  "spoofed",
		"first_event_at":           "caller-owned",
		"runtime_language":         "go",
	})

	capture := captureFrom(t, fake)
	if capture.Properties["aggregate_first_event_at"] != first {
		t.Errorf("expected generated aggregate_first_event_at %v, got %v", first, capture.Properties["aggregate_first_event_at"])
	}
	if capture.Properties["aggregate_last_event_at"] != last {
		t.Errorf("expected generated aggregate_last_event_at %v, got %v", last, capture.Properties["aggregate_last_event_at"])
	}
	if capture.Properties["count"] != int64(5) {
		t.Errorf("expected count 5, got %v", capture.Properties["count"])
	}
	if capture.Properties["first_event_at"] != "caller-owned" {
		t.Errorf("expected caller-owned first_event_at to pass through, got %v", capture.Properties["first_event_at"])
	}
	if capture.Properties["runtime_language"] != "go" {
		t.Errorf("expected runtime_language to pass through, got %v", capture.Properties["runtime_language"])
	}
}

func TestFlushCount_UnknownEventTimesOmitted(t *testing.T) {
	fake := &fakeClient{}
	p := newTestAnalytics(fake)

	p.flushCount(analytics.Worker, analytics.Register, uuid.New(), nil, 1, time.Time{}, time.Time{}, analytics.Properties{
		"aggregate_first_event_at": "spoofed",
		"aggregate_last_event_at":  "spoofed",
	})

	capture := captureFrom(t, fake)
	if v, ok := capture.Properties["aggregate_first_event_at"]; ok {
		t.Errorf("expected no aggregate_first_event_at, got %v", v)
	}
	if v, ok := capture.Properties["aggregate_last_event_at"]; ok {
		t.Errorf("expected no aggregate_last_event_at, got %v", v)
	}
}
