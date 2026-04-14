package v1

import "time"

const defaultRetention = 720 * time.Hour

func retentionBoundary(retentionPeriod string) time.Time {
	retention, err := time.ParseDuration(retentionPeriod)
	if err != nil || retention <= 0 {
		retention = defaultRetention
	}

	return time.Now().Add(-retention)
}

// IsBeforeRetention returns true when the given timestamp is older than the
// tenant's retention window (now - retentionPeriod).
func IsBeforeRetention(t time.Time, retentionPeriod string) bool {
	if t.IsZero() {
		return false
	}

	return t.Before(retentionBoundary(retentionPeriod))
}
