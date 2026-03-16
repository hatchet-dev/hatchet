package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func ValidateJSONB(jsonb []byte, fieldName string) error {
	if len(jsonb) == 0 {
		return nil
	}

	if !isUnicodeValid(jsonb) {
		return fmt.Errorf("encoded jsonb contains invalid null character \\u0000 in field `%s`", fieldName)
	}

	if !json.Valid(jsonb) {
		return fmt.Errorf("invalid json in field `%s`", fieldName)
	}

	return nil
}

func isUnicodeValid(jsonb []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(jsonb))
	for {
		token, err := dec.Token()
		if err != nil {
			// NOTE(gregfurman): regardless of whether io.EOF or actual parsing error,
			// just return early as json.Valid should catch invalid payload.
			return true
		}
		if s, ok := token.(string); ok {
			if strings.ContainsRune(s, '\u0000') {
				return false
			}
		}
	}
}
