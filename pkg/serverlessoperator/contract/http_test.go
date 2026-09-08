//go:build !e2e && !load && !rampup && !integration

package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyNamespace(t *testing.T) {
	const ns = "11111111-2222-3333-4444-555555555555"

	assert.Equal(t, ns+"_wf", ApplyNamespace(ns, "wf"))
	assert.Equal(t, ns+"_wf", ApplyNamespace(ns, ns+"_wf"), "already prefixed names are left alone")
	assert.Equal(t, ns+"_"+ns+"wf", ApplyNamespace(ns, ns+"wf"), "the separator is part of the prefix")
	assert.Equal(t, "wf", ApplyNamespace("", "wf"), "no namespace applies nothing")
	assert.Equal(t, ns+"_", ApplyNamespace(ns, ""))
	assert.Equal(t, ns+"_", NamespacePrefix(ns))
}
