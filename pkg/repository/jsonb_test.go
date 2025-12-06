//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateJSONB(t *testing.T) {
	tests := []struct {
		name  string
		jsonb []byte
		valid bool
	}{
		{
			name:  "valid_jsonb",
			jsonb: []byte(`{"int_zero":0, "string_zero":"0", "null":null, "null_string":"", "normal_string": "Hello 你好 🙂"}`),
			valid: true,
		},
		{
			name:  "valid_jsonb_with_encoded_NUL",
			jsonb: []byte("{\"NUL\":\"\\u0000\"}"),
			valid: true,
		},
		{
			name:  "invalid_jsonb_with_unicode_NUL",
			jsonb: []byte("{\"NUL\":\"\u0000\"}"),
			valid: false,
		},
		{
			name:  "invalid_jsonb_with_utf8_NUL",
			jsonb: []byte("{\"NUL\":\"\x00\"}"),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONB(tt.jsonb, tt.name)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
