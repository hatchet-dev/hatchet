package repository

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ValidateJSONB rejects JSON that Postgres jsonb cannot store and returns the bytes to store in
// place of the input. They differ only where the input has an unpaired UTF-16 surrogate escape,
// which Postgres rejects; that escape becomes \ufffd, the character Go's decoder yields for it.
func ValidateJSONB[T string | []byte](jsonb T, fieldName string) (T, error) {
	if len(jsonb) == 0 {
		return jsonb, nil
	}

	b := []byte(jsonb)

	if !IsUnicodeValid(b) {
		return jsonb, fmt.Errorf("encoded jsonb contains invalid null character \\u0000 in field `%s`", fieldName)
	}

	if !json.Valid(b) {
		return jsonb, fmt.Errorf("invalid json in field `%s`", fieldName)
	}

	if replaced := replaceUnpairedSurrogateEscapes(b); replaced != nil {
		return T(replaced), nil
	}

	return jsonb, nil
}

var unicodeEscapePrefix = []byte(`\u`)

// replaceUnpairedSurrogateEscapes expects valid JSON and returns a copy with every unpaired
// surrogate escape replaced, or nil if there is none.
func replaceUnpairedSurrogateEscapes(b []byte) []byte {
	if !bytes.Contains(b, unicodeEscapePrefix) {
		return nil
	}

	var out []byte

	for i := 0; i < len(b); i++ {
		if b[i] != '\\' {
			continue
		}

		// valid JSON puts an escape character after every backslash and four hex digits after \u
		i++

		if b[i] != 'u' {
			continue
		}

		start := i - 1
		r := decodeEscapedRune(b[i+1 : i+5])
		i += 4

		if !utf16.IsSurrogate(r) {
			continue
		}

		if r < 0xdc00 && i+6 < len(b) && b[i+1] == '\\' && b[i+2] == 'u' {
			if low := decodeEscapedRune(b[i+3 : i+7]); low >= 0xdc00 && low <= 0xdfff {
				i += 6
				continue
			}
		}

		if out == nil {
			out = bytes.Clone(b)
		}

		copy(out[start:], `\ufffd`)
	}

	return out
}

func decodeEscapedRune(hexDigits []byte) rune {
	var buf [2]byte

	if _, err := hex.Decode(buf[:], hexDigits); err != nil {
		return utf8.RuneError
	}

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
