package apitokens

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCapabilities(t *testing.T) {
	cases := []struct {
		name string
		caps []string
		ok   bool
	}{
		{"known capabilities", []string{"worker", "events:push"}, true},
		{"read preset", []string{"read-only"}, true},
		{"unknown capability", []string{"worker", "workers:fly"}, false},
		{"empty list", []string{}, false},
		{"all", []string{"all"}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			apiErr, err := validateCapabilities(c.caps)
			require.NoError(t, err)
			require.Equal(t, c.ok, apiErr == nil)
		})
	}
}
