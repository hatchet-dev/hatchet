//go:build !e2e && !load && !rampup && !integration

package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestApplyAuthDisabledOverrides(t *testing.T) {
	rt := &server.ConfigFileRuntime{
		AllowCreateTenant: true,
		AllowSignup:       true,
		AllowInvites:      true,
	}

	applyAuthDisabledOverrides(rt)

	assert.False(t, rt.AllowCreateTenant, "authdisabled must disable tenant creation")
	assert.False(t, rt.AllowSignup, "authdisabled must disable signup")
	assert.False(t, rt.AllowInvites, "authdisabled must disable invites")
}
