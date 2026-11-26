package worker

import (
	"errors"
	"time"
)

// Deprecated: use hatchet.NonRetryableError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type NonRetryableError struct {
	e error
}

func (e *NonRetryableError) Error() string {
	return e.e.Error()
}

// Deprecated: use hatchet.NewNonRetryableError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewNonRetryableError(err error) error {
	return &NonRetryableError{e: err}
}

// Deprecated: use hatchet.IsNonRetryableError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func IsNonRetryableError(err error) bool {
	e := &NonRetryableError{}
	return errors.As(err, &e)
}

// Deprecated: use hatchet.RetryAfterError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type RetryAfterError struct {
	e     error
	After time.Duration
}

func (e *RetryAfterError) Error() string {
	return e.e.Error()
}

func (e *RetryAfterError) Unwrap() error {
	return e.e
}

// Deprecated: use hatchet.NewRetryAfterError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewRetryAfterError(after time.Duration, err error) error {
	return &RetryAfterError{e: err, After: after}
}

// Deprecated: use hatchet.AsRetryAfterError from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func AsRetryAfterError(err error) (*RetryAfterError, bool) {
	e := &RetryAfterError{}
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// ErrTaskCancelled is the cause of a task's context when the run is cancelled,
// for example by a user or by the engine. Read it with context.Cause; ctx.Err()
// still returns context.Canceled.
//
// Deprecated: use hatchet.ErrTaskCancelled from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
var ErrTaskCancelled = errors.New("task cancelled")

// ErrTaskEvicted is the cause of a durable task's context when the worker evicts
// the run (TTL, capacity pressure, a server request or worker shutdown). The run
// is not cancelled: it resumes on a worker later. Read it with context.Cause;
// ctx.Err() still returns context.Canceled.
//
// Deprecated: use hatchet.ErrTaskEvicted from github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
var ErrTaskEvicted = errors.New("task evicted")
