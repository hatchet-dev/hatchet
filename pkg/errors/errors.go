package errors

import (
	"fmt"
)

type DetailedError struct {
	Reason      string `json:"reason"`
	Description string `json:"description"`
	DocsLink    string `json:"docs_link"`
	Code        uint   `json:"code"`
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
