package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cleanAdditionalMetadataTableTest(t *testing.T) {

	tests := []struct {
		name               string
		additionalMetadata []byte
		expected           map[string]interface{}
	}{

		{
			name:               "empty",
			additionalMetadata: []byte(""),
			expected:           map[string]interface{}{},
		},
		{
			name:               "null",
			additionalMetadata: []byte("null"),
			expected:           map[string]interface{}{},
		},
		{
			name:               "valid",
			additionalMetadata: []byte(`{"key":"value"}`),
			expected:           map[string]interface{}{"key": "value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := cleanAdditionalMetadata(test.additionalMetadata)
			assert.Equal(t, test.expected, actual)
		})
	}
}
