package defaults

import "time"

const (
	DefaultJobRunTimeout  = "60m"
	DefaultStepRunTimeout = "300s"

	DefaultScheduleTimeout = 5 * time.Minute
)
