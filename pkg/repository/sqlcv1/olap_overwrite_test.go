//go:build !e2e && !load && !rampup && !integration

package sqlcv1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredWorkflowIdsVariants(t *testing.T) {
	assert.Len(t, requiredWorkflowIdsVariants, 6)

	for query, variant := range requiredWorkflowIdsVariants {
		assert.NotEqual(t, query, variant, "optional workflow_id filter not found in query")
		assert.NotContains(t, variant, "$3::uuid[] IS NULL")
		assert.Contains(t, variant, "workflow_id = ANY($3::uuid[])")
	}
}
