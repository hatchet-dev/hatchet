package loaderutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHeaders(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "single pair",
			input: "X-API-Key=foo",
			want:  map[string]string{"X-API-Key": "foo"},
		},
		{
			name:  "multiple pairs",
			input: "X-API-Key=foo,X-Tenant=bar",
			want:  map[string]string{"X-API-Key": "foo", "X-Tenant": "bar"},
		},
		{
			name:  "value contains equals",
			input: "X-Token=abc=def",
			want:  map[string]string{"X-Token": "abc=def"},
		},
		{
			name:  "whitespace trimmed",
			input: "  X-API-Key = foo , X-Tenant = bar  ",
			want:  map[string]string{"X-API-Key": "foo", "X-Tenant": "bar"},
		},
		{
			name:  "empty value allowed",
			input: "X-Empty=",
			want:  map[string]string{"X-Empty": ""},
		},
		{
			name:  "trailing comma skipped",
			input: "X-A=1,X-B=2,",
			want:  map[string]string{"X-A": "1", "X-B": "2"},
		},
		{
			name:    "missing equals",
			input:   "X-API-Key",
			wantErr: true,
		},
		{
			name:    "empty key",
			input:   "=value",
			wantErr: true,
		},
		{
			name:    "whitespace-only key",
			input:   "   =value",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHeaders(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
