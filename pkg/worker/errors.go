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
