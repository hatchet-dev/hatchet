package worker

import "errors"

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
