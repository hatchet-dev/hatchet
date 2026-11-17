package loader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewPgxPoolConfig_ConnectBounds(t *testing.T) {
	cases := []struct {
		name string
		url  string
		opts PgxPoolOpts
		want time.Duration
	}{
		{
			name: "defaults",
			url:  "postgres://user@127.0.0.1:5432/db",
			want: defaultPgxConnectTimeout,
		},
		{
			name: "overrides connect_timeout in the URL",
			url:  "postgres://user@127.0.0.1:5432/db?connect_timeout=120",
			want: defaultPgxConnectTimeout,
		},
		{
			name: "explicit",
			url:  "postgres://user@127.0.0.1:5432/db",
			opts: PgxPoolOpts{ConnectTimeout: 7 * time.Second},
			want: 7 * time.Second,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := NewPgxPoolConfig(tc.url, tc.opts)
			require.NoError(t, err)

			require.Equal(t, tc.want, cfg.ConnConfig.ConnectTimeout)
			require.NotNil(t, cfg.ConnConfig.DialFunc)
		})
	}
}
