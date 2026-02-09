package repository

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateJSONB(jsonb []byte, fieldName string) error {
	if len(jsonb) == 0 {
		return nil
	}

	if strings.Contains(string(jsonb), "\\u0000") {
		return fmt.Errorf("encoded jsonb contains invalid null character \\u0000 in field `%s`", fieldName)
	}

	if !json.Valid(jsonb) {
		return fmt.Errorf("invalid json in field `%s`", fieldName)
	}

	return nil
}
