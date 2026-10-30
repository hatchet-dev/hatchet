package v1

import (
	"testing"
	"time"
)

func TestIsBeforeRetention(t *testing.T) {
	t.Parallel()

	if IsBeforeRetention(time.Time{}, "720h") {
		t.Fatal("zero time should not be before retention")
	}

	if IsBeforeRetention(time.Now(), "24h") {
		t.Fatal("now should be inside a 24h window")
	}

	if !IsBeforeRetention(time.Now().Add(-48*time.Hour), "24h") {
		t.Fatal("48h ago should be before a 24h window")
	}
}

func TestRetentionBoundary(t *testing.T) {
	t.Parallel()

	if got := RetentionBoundary("24h"); time.Since(got) < 24*time.Hour || time.Since(got) > 24*time.Hour+time.Minute {
		t.Fatalf("24h boundary is %s ago", time.Since(got))
	}

	if got := RetentionBoundary("not a duration"); time.Since(got) < 720*time.Hour || time.Since(got) > 720*time.Hour+time.Minute {
		t.Fatalf("an unparseable period should fall back to 720h, got %s ago", time.Since(got))
	}
}
