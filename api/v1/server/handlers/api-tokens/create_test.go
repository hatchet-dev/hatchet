package apitokens

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCapabilities(t *testing.T) {
	cases := []struct {
		name     string
		caps     []string
		readOnly bool
		ok       bool
	}{
		{"known capabilities", []string{"worker", "events:push"}, false, true},
		{"all", []string{"all"}, false, true},
		{"unknown capability", []string{"worker", "workers:fly"}, false, false},
		{"empty list", []string{}, false, false},
		{"read-only token", []string{"worker"}, true, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			apiErr, err := validateCapabilities(c.caps, c.readOnly)
			require.NoError(t, err)
			require.Equal(t, c.ok, apiErr == nil)
		})
	}
}
