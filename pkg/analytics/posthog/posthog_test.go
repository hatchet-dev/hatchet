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
		"first_event_at":   "spoofed",
		"last_event_at":    "spoofed",
		"runtime_language": "go",
	})

	capture := captureFrom(t, fake)
	if capture.Properties["first_event_at"] != first {
		t.Errorf("expected generated first_event_at %v, got %v", first, capture.Properties["first_event_at"])
	}
	if capture.Properties["last_event_at"] != last {
		t.Errorf("expected generated last_event_at %v, got %v", last, capture.Properties["last_event_at"])
	}
	if capture.Properties["count"] != int64(5) {
		t.Errorf("expected count 5, got %v", capture.Properties["count"])
	}
	if capture.Properties["runtime_language"] != "go" {
		t.Errorf("expected runtime_language to pass through, got %v", capture.Properties["runtime_language"])
	}
}

func TestFlushCount_UnknownEventTimesOmitted(t *testing.T) {
	fake := &fakeClient{}
	p := newTestAnalytics(fake)

	p.flushCount(analytics.Worker, analytics.Register, uuid.New(), nil, 1, time.Time{}, time.Time{}, analytics.Properties{
		"first_event_at": "spoofed",
		"last_event_at":  "spoofed",
	})

	capture := captureFrom(t, fake)
	if v, ok := capture.Properties["first_event_at"]; ok {
		t.Errorf("expected no first_event_at, got %v", v)
	}
	if v, ok := capture.Properties["last_event_at"]; ok {
		t.Errorf("expected no last_event_at, got %v", v)
	}
}
