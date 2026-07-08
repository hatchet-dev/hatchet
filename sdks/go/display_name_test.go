package hatchet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRunDisplayName_setsRunOpt(t *testing.T) {
	opts := &runOpts{}

	WithRunDisplayName("Acme Corp")(opts)

	require.NotNil(t, opts.DisplayName)
	assert.Equal(t, "Acme Corp", *opts.DisplayName)
}

func TestRunOpts_displayNameUnsetByDefault(t *testing.T) {
	opts := &runOpts{}

	for _, opt := range []RunOptFunc{WithRunKey("k"), WithRunSticky(true)} {
		opt(opts)
	}

	assert.Nil(t, opts.DisplayName, "display name stays unset unless WithRunDisplayName is used")
}
