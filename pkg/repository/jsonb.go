package repository

import (
	"bytes"
	"fmt"
)

func ValidateJSONB(jsonb []byte, fieldName string) error {
	if len(jsonb) == 0 {
		return nil
	}

	if bytes.Contains(jsonb, []byte("\u0000")) {
		return fmt.Errorf("encoded jsonb contains invalid null character \\u0000 in field `%s`", fieldName)
	}

	return nil
}
