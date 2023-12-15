package types

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Action struct {
	// Required. The integration that this uses.
	IntegrationID string

	// Required. The verb to perform.
	Verb string

	// Optional. A way to unique identify the step.
	Subresource string
}

func (o Action) String() string {
	if o.Subresource != "" {
		return fmt.Sprintf("%s:%s:%s", o.IntegrationID, o.Verb, o.Subresource)
	}

	return o.IntegrationVerbString()
}

func (o Action) IntegrationVerbString() string {
	return fmt.Sprintf("%s:%s", o.IntegrationID, o.Verb)
}

// ParseActionID parses an action ID into its constituent parts.
func ParseActionID(actionID string) (Action, error) {
	parts := strings.Split(actionID, ":")
	numParts := len(parts)

	integrationId := firstToLower(parts[0])
	verb := strings.ToLower(parts[1])

	var subresource string
	if numParts == 3 {
		subresource = firstToLower(parts[2])
	}

	return Action{
		IntegrationID: integrationId,
		Verb:          verb,
		Subresource:   subresource,
	}, nil
}

// source: https://stackoverflow.com/questions/75988064/make-first-letter-of-string-lower-case-in-golang
func firstToLower(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}
