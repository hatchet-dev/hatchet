//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChildKeysMatchParentKeys(t *testing.T) {
	assert.True(t, childKeysMatchParentKeys("input.case_number"))
	assert.True(t, childKeysMatchParentKeys(`additional_metadata.tenant + ":" + input.user`))
	assert.False(t, childKeysMatchParentKeys("workflow_run_id"))
	assert.False(t, childKeysMatchParentKeys(`workflow_run_id.substring(0, 1)`))
	assert.False(t, childKeysMatchParentKeys(`parents.step1.key`))
}
