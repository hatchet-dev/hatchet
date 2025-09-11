package worker

import "errors"

type NonRetryableError struct {
	e error
}

func (e *NonRetryableError) Error() string {
	return e.e.Error()
}

func NewNonRetryableError(err error) error {
	return &NonRetryableError{e: err}
}

func IsNonRetryableError(err error) bool {
	e := &NonRetryableError{}
	return errors.As(err, &e)
}
