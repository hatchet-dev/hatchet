package repository

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
)

func ValidateJSONB(jsonb []byte, fieldName string) error {
	if len(jsonb) == 0 {
		return nil
	}

	if !IsUnicodeValid(jsonb) {
		return fmt.Errorf("encoded jsonb contains invalid null character \\u0000 in field `%s`", fieldName)
	}

	if !json.Valid(jsonb) {
		return fmt.Errorf("invalid json in field `%s`", fieldName)
	}

	if hasUnpairedSurrogateEscape(jsonb) {
		return fmt.Errorf("json contains an unpaired UTF-16 surrogate escape, such as half of an emoji, in field `%s`", fieldName)
	}

	return nil
}

var unicodeEscapePrefix = []byte(`\u`)

// hasUnpairedSurrogateEscape expects valid JSON.
//
// A character outside the Basic Multilingual Plane, such as an emoji, is a pair of UTF-16
// surrogates, and JSON can write it as two \u escapes. The JSON grammar also allows one of the two
// alone, and Go accepts that. Postgres jsonb rejects it, because half a pair has no UTF-8 encoding.
// JavaScript's JSON.stringify and Python's json.dumps write half a pair for a string that holds
// one, which in JavaScript is what text.slice(0, n) leaves when it cuts through an emoji.
func hasUnpairedSurrogateEscape(b []byte) bool {
	if !bytes.Contains(b, unicodeEscapePrefix) {
		return false
	}

	for i := 0; i < len(b); i++ {
		if b[i] != '\\' {
			continue
		}

		// valid JSON puts an escape character after every backslash and four hex digits after \u
		i++

		if b[i] != 'u' {
			continue
		}

		r := decodeEscapedRune(b[i+1 : i+5])
		i += 4

		if !utf16.IsSurrogate(r) {
			continue
		}

		// a pair is a high surrogate escape immediately followed by a low one
		if r >= 0xdc00 || i+6 >= len(b) || b[i+1] != '\\' || b[i+2] != 'u' {
			return true
		}

		if low := decodeEscapedRune(b[i+3 : i+7]); low < 0xdc00 || low > 0xdfff {
			return true
		}

		i += 6
	}

	return false
}

func decodeEscapedRune(hexDigits []byte) rune {
	var buf [2]byte

	// the caller passes four hex digits
	_, _ = hex.Decode(buf[:], hexDigits)

	return rune(buf[0])<<8 | rune(buf[1])
}

func IsUnicodeValid(jsonb []byte) bool {
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
