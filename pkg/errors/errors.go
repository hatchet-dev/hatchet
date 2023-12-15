package errors

import (
	"fmt"
)

type DetailedError struct {
	// a custom Hatchet error code
	// example: 1400
	Code uint `json:"code"`

	// a reason for this error
	Reason string `json:"reason"`

	// a description for this error
	// example: A descriptive error message
	Description string `json:"description"`

	// a link to the documentation for this error, if it exists
	// example: github.com/hatchet-dev/hatchet
	DocsLink string `json:"docs_link"`
}

func (e DetailedError) Error() string {
	errStr := fmt.Sprintf("error %d: %s", e.Code, e.Description)

	if e.DocsLink != "" {
		errStr = fmt.Sprintf("%s, see %s", errStr, e.DocsLink)
	}

	return errStr
}

func NewError(code uint, reason, description, docsLink string) *DetailedError {
	return &DetailedError{
		Code:        code,
		Reason:      reason,
		Description: description,
		DocsLink:    docsLink,
	}
}

func NewErrInternal(err error) *DetailedError {
	return NewError(500, "Internal Server Error", err.Error(), "")
}

func NewErrForbidden(err error) *DetailedError {
	return NewError(403, "Forbidden", err.Error(), "")
}
