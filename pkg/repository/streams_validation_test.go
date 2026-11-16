package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStreamAddress(t *testing.T) {
	longest := strings.Repeat("😀", MaxStreamNameLength)
	tooLong := strings.Repeat("a", MaxStreamNameLength+1)

	tests := []struct {
		name      string
		namespace string
		topic     string
		wantErr   string
	}{
		{name: "default namespace", topic: "orders"},
		{name: "limit counts characters, not bytes", namespace: longest, topic: longest},
		{name: "missing topic", namespace: "ns", wantErr: "topic is required"},
		{name: "topic too long", topic: tooLong, wantErr: "topic is 256 characters long"},
		{name: "namespace too long", namespace: tooLong, topic: "orders", wantErr: "namespace is 256 characters long"},
		{name: "NUL in topic", topic: "ord\x00ers", wantErr: "topic must not contain NUL"},
		{name: "NUL in namespace", namespace: "n\x00s", topic: "orders", wantErr: "namespace must not contain NUL"},
		{name: "invalid UTF-8", topic: "\xff\xfe", wantErr: "topic must be valid UTF-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStreamAddress(tt.namespace, tt.topic)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestStreamPayloadMessageUnits(t *testing.T) {
	assert.Equal(t, int32(1), StreamPayloadMessageUnits(1))
	assert.Equal(t, int32(1), StreamPayloadMessageUnits(MaxStreamMessagePayloadBytes))
	assert.Equal(t, int32(2), StreamPayloadMessageUnits(MaxStreamMessagePayloadBytes+1))
	assert.Equal(t, int32(17), StreamPayloadMessageUnits(MaxStreamUploadedPayloadBytes))
}
