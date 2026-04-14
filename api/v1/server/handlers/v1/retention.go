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

// ClampToRetention ensures the requested time is not earlier than the tenant's
// retention boundary (now - retentionPeriod).
func ClampToRetention(requested time.Time, retentionPeriod string) time.Time {
	boundary := retentionBoundary(retentionPeriod)

	if requested.IsZero() || requested.Before(boundary) {
		return boundary
	}

	return requested
}

// ClampToRetentionPtr is like ClampToRetention but accepts a pointer. A nil
// pointer is treated as "no constraint" and gets clamped to the boundary.
func ClampToRetentionPtr(requested *time.Time, retentionPeriod string) time.Time {
	boundary := retentionBoundary(retentionPeriod)

	if requested == nil || requested.Before(boundary) {
		return boundary
	}

	return *requested
}

// IsBeforeRetention returns true when the given timestamp is older than the
// tenant's retention window (now - retentionPeriod).
func IsBeforeRetention(t time.Time, retentionPeriod string) bool {
	if t.IsZero() {
		return false
	}

	return t.Before(retentionBoundary(retentionPeriod))
}
