package types

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Action struct {
	// Required. The service that this belongs to
	Service string

	// Required. The verb to perform.
	Verb string

	// Optional. A way to unique identify the step.
	Subresource string
}

func (o Action) String() string {
	if o.Subresource != "" {
		return fmt.Sprintf("%s:%s:%s", o.Service, o.Verb, o.Subresource)
	}

	return o.IntegrationVerbString()
}

func (o Action) IntegrationVerbString() string {
	return fmt.Sprintf("%s:%s", o.Service, o.Verb)
}

// ParseActionID parses an action ID into its constituent parts.
func ParseActionID(actionID string) (Action, error) {
	parts := strings.Split(actionID, ":")
	numParts := len(parts)

	if numParts < 2 || numParts > 3 {
		return Action{}, fmt.Errorf("invalid action id %s, must have at least 2 strings separated : (colon)", actionID)
	}

	Service := firstToLower(parts[0])
	verb := strings.ToLower(parts[1])

	var subresource string
	if numParts == 3 {
		subresource = firstToLower(parts[2])
	}

	return Action{
		Service:     Service,
		Verb:        verb,
		Subresource: subresource,
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
