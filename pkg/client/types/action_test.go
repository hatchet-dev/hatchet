//go:build !e2e && !load && !rampup && !integration

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseActionID(t *testing.T) {
	action, err := ParseActionID("Svc:Run")
	require.NoError(t, err)
	assert.Equal(t, Action{Service: "svc", Verb: "run"}, action)

	action, err = ParseActionID("svc:run:Sub")
	require.NoError(t, err)
	assert.Equal(t, Action{Service: "svc", Verb: "run", Subresource: "sub"}, action)

	_, err = ParseActionID("run")
	assert.Error(t, err, "a single component")

	_, err = ParseActionID("a:b:c:d")
	assert.Error(t, err, "four components")
}

// A semicolon is the separator the worker action hash frames ids with, so it is refused
// wherever it appears, including where the colon split would otherwise accept the id.
func TestParseActionIDRejectsSemicolons(t *testing.T) {
	for _, id := range []string{"svc:a;svc:b", "svc:run;", ";svc:run", "svc;x:run", "svc:run:sub;"} {
		_, err := ParseActionID(id)
		assert.ErrorContains(t, err, "semicolon", id)
	}
}
