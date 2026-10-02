package nats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

func TestIsStale(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		kind        msgqueue.TopicKind
		publishedAt time.Time
		wantAge     time.Duration
		wantStale   bool
	}{
		{"scheduler fresh", msgqueue.TopicKindSchedulerPartition, now.Add(-time.Second), time.Second, false},
		{"scheduler at max age", msgqueue.TopicKindSchedulerPartition, now.Add(-5 * time.Second), 5 * time.Second, false},
		{"scheduler past max age", msgqueue.TopicKindSchedulerPartition, now.Add(-6 * time.Second), 6 * time.Second, true},
		{"tenant stream past scheduler max age", msgqueue.TopicKindTenantStream, now.Add(-6 * time.Second), 6 * time.Second, false},
		{"tenant stream past max age", msgqueue.TopicKindTenantStream, now.Add(-31 * time.Second), 31 * time.Second, true},
		{"unstamped", msgqueue.TopicKindSchedulerPartition, time.Time{}, 0, false},
		{"stamped in the future", msgqueue.TopicKindSchedulerPartition, now.Add(time.Minute), -time.Minute, false},
		{"unknown topic kind", msgqueue.TopicKind("other"), now.Add(-time.Hour), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, stale := isStale(tt.kind, tt.publishedAt, now)
			assert.Equal(t, tt.wantStale, stale)
			assert.Equal(t, tt.wantAge, age)
		})
	}
}