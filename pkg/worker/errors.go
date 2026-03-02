package worker

import "errors"

// Deprecated: NonRetryableError is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type NonRetryableError struct {
	e error
}

// Deprecated: Error is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (e *NonRetryableError) Error() string {
	return e.e.Error()
}

// Deprecated: NewNonRetryableError is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewNonRetryableError(err error) error {
	return &NonRetryableError{e: err}
}

// Deprecated: IsNonRetryableError is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func IsNonRetryableError(err error) bool {
	e := &NonRetryableError{}
	return errors.As(err, &e)
}
