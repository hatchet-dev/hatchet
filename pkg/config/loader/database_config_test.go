package loader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDatabaseConfigFile_AcquireTimeout(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{name: "default", want: 30 * time.Second},
		{name: "explicit", env: "5s", want: 5 * time.Second},
		{name: "zero disables the bound", env: "0", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("DATABASE_ACQUIRE_TIMEOUT", tc.env)
			}

			cf, err := LoadDatabaseConfigFile()
			require.NoError(t, err)
			require.Equal(t, tc.want, cf.AcquireTimeout)
		})
	}
}
