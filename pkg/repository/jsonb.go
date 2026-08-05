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

// nullEscape is the JSON escape sequence Postgres rejects with SQLSTATE 22P05
// ("unsupported Unicode escape sequence") when casting to jsonb. JSON only
// allows a lowercase `u` here and `0000` has no hex letters, so a plain byte
// comparison covers every way it can be spelled.
var nullEscape = []byte{'\\', 'u', '0', '0', '0', '0'}

// SanitizeJSONB strips the characters Postgres refuses to store in a jsonb
// column: `\u0000` escapes and raw NUL bytes. Both come in from data we don't
// control (worker error messages, task outputs), and internal write paths like
// the payload store can't reject them without wedging the queue consumer that
// is trying to write them.
//
// The input is returned untouched when there is nothing to strip, so the common
// case is one scan and no allocation.
func SanitizeJSONB(jsonb []byte) []byte {
	if !bytes.Contains(jsonb, nullEscape) && bytes.IndexByte(jsonb, 0) < 0 {
		return jsonb
	}

	out := make([]byte, 0, len(jsonb))
	inString := false

	for i := 0; i < len(jsonb); {
		c := jsonb[i]

		switch {
		case c == 0:
			// a raw NUL byte is never valid inside a JSON document
			i++
		case !inString:
			if c == '"' {
				inString = true
			}

			out = append(out, c)
			i++
		case c == '\\':
			// consume the escape sequence as a unit, otherwise an escaped
			// backslash would make us misread the bytes after it as an escape
			// of their own
			if i+1 >= len(jsonb) {
				out = append(out, c)
				i++

				continue
			}

			if i+len(nullEscape) <= len(jsonb) && bytes.Equal(jsonb[i:i+len(nullEscape)], nullEscape) {
				i += len(nullEscape)

				continue
			}

			out = append(out, c, jsonb[i+1])
			i += 2
		case c == '"':
			inString = false

			out = append(out, c)
			i++
		default:
			out = append(out, c)
			i++
		}
	}

	return out
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
