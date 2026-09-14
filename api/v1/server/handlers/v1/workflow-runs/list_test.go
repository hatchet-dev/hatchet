package workflowruns

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestAdditionalMetadataOperator(t *testing.T) {
	andOp := gen.AND
	orOp := gen.OR

	tests := []struct {
		name                 string
		param                *gen.V1AdditionalMetadataOperator
		numFilters           int
		useGinIndexOverride  bool
		hasDuplicateKeys     bool
		expected             v1.AdditionalMetadataOperator
	}{
		{
			name:                 "single filter defaults to AND for index optimization",
			param:                nil,
			numFilters:           1,
			useGinIndexOverride:  false,
			hasDuplicateKeys:     false,
			expected:             v1.AdditionalMetadataOperatorAnd,
		},
		{
			name:                 "multiple filters default to OR",
			param:                nil,
			numFilters:           2,
			useGinIndexOverride:  false,
			hasDuplicateKeys:     false,
			expected:             v1.AdditionalMetadataOperatorOr,
		},
		{
			name:                 "multiple filters with explicit AND",
			param:                &andOp,
			numFilters:           2,
			useGinIndexOverride:  false,
			hasDuplicateKeys:     false,
			expected:             v1.AdditionalMetadataOperatorAnd,
		},
		{
			name:                 "multiple filters with explicit OR",
			param:                &orOp,
			numFilters:           2,
			useGinIndexOverride:  false,
			hasDuplicateKeys:     false,
			expected:             v1.AdditionalMetadataOperatorOr,
		},
		{
			name:                 "multiple filters with gin index override without duplicates uses AND",
			param:                nil,
			numFilters:           2,
			useGinIndexOverride:  true,
			hasDuplicateKeys:     false,
			expected:             v1.AdditionalMetadataOperatorAnd,
		},
		{
			name:                 "duplicate keys default to OR semantics",
			param:                nil,
			numFilters:           2,
			useGinIndexOverride:  false,
			hasDuplicateKeys:     true,
			expected:             v1.AdditionalMetadataOperatorOr,
		},
		{
			name:                 "duplicate keys with gin index override still uses OR semantics",
			param:                nil,
			numFilters:           2,
			useGinIndexOverride:  true,
			hasDuplicateKeys:     true,
			expected:             v1.AdditionalMetadataOperatorOr,
		},
		{
			name:                 "duplicate keys with explicit OR still uses OR semantics",
			param:                &orOp,
			numFilters:           2,
			useGinIndexOverride:  true,
			hasDuplicateKeys:     true,
			expected:             v1.AdditionalMetadataOperatorOr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := additionalMetadataOperator(tc.param, tc.numFilters, tc.useGinIndexOverride, tc.hasDuplicateKeys)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
