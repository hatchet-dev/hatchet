package scheduling

import "time"

func IsTimedout(qi *QueueItemWithOrder) bool {
	// if the current time is after the scheduleTimeoutAt, then mark this as timed out
	now := time.Now().UTC().UTC()
	scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

	// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
	isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

	return isTimedOut
}
