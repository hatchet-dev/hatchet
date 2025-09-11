//go:build !e2e && !load && !rampup && !integration

package signature

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
				actual, err := Sign("hello world", "secret")
				if err != nil {
					t.Fatal(err)
				}

				expected := "734cc62f32841568f45715aeb9f4d7891324e6d948e4c6c60c0621cdac48623a"

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
