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
