//go:build !e2e && !load && !rampup && !integration

package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignature(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "should hash",
			run: func(t *testing.T) {
				actual, err := Sign("hello world", "test-signing-secret-0123456789abcdef")
				if err != nil {
					t.Fatal(err)
				}

				expected := "2b4d2a1858d8dbcc64653ea5463be91d97665406c46a12cd5ebdeb92cbe63a05"

				assert.Equal(t, expected, actual)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func TestVerify(t *testing.T) {
	sig, err := Sign("hello world", "test-signing-secret-0123456789abcdef")
	assert.NoError(t, err)

	assert.True(t, Verify("hello world", "test-signing-secret-0123456789abcdef", sig))
	assert.False(t, Verify("hello world", "test-signing-secret-0123456789abcdef", "deadbeef"))
	assert.False(t, Verify("hello world", "wrong-signing-secret-0123456789abcdef", sig))
	assert.False(t, Verify("tampered", "test-signing-secret-0123456789abcdef", sig))
}
