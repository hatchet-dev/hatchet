package random

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name    string
		args    args
		want    func(string) bool
		wantErr assert.ErrorAssertionFunc
	}{{
		name: "GenerateRandomString",
		args: args{
			n: 32,
		},
		want: func(s string) bool {
			if match, err := regexp.MatchString(`^[0-9a-zA-Z]+$`, s); err != nil || !match {
				return false
			}
			return len(s) == 32
		},
		wantErr: assert.NoError,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.args.n)
			if !tt.wantErr(t, err, fmt.Sprintf("GenerateRandomString(%v)", tt.args.n)) {
				return
			}
			assert.Equalf(t, true, tt.want(got), "GenerateRandomString(%v)", tt.args.n)
		})
	}
}

func TestGenerateWebhookSecret(t *testing.T) {
	s, err := GenerateWebhookSecret()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equalf(t, 32, len(s), "GenerateWebhookSecret length should be 32")
}
