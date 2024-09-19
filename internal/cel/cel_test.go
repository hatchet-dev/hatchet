package cel_test

import (
	"testing"

	"github.com/google/cel-go/common/types"

	"github.com/hatchet-dev/hatchet/internal/cel"

	"github.com/stretchr/testify/assert"
)

func TestCELParser(t *testing.T) {
	parser := cel.NewCELParser()

	tests := []struct {
		expression  string
		input       cel.WorkflowStringInput
		expected    string
		expectError bool
	}{
		{
			expression: `has(input.custom.value) ? input.custom.value : "default"`,
			input: cel.NewWorkflowStringInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "actual value",
					},
				}),
			),
			expected:    "actual value",
			expectError: false,
		},
		{
			expression: `has(input.custom) ? input.custom.value : "default"`,
			input: cel.NewWorkflowStringInput(
				cel.WithInput(map[string]interface{}{}),
			),
			expected:    "default",
			expectError: false,
		},
		{
			expression: `checksum(input.custom.value)`,
			input: cel.NewWorkflowStringInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "checksum this",
					},
				}),
			),
			expected:    types.String("97e9269cd0514f864e6be9157998464c94776ebc7f669b449f581abdad4035f5").Value().(string), // Precomputed checksum
			expectError: false,
		},
		{
			expression: `input.custom.value + workflow_run_id`,
			input: cel.NewWorkflowStringInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "concatenate ",
					},
				}),
				cel.WithWorkflowRunID("1234"),
			),
			expected:    "concatenate 1234",
			expectError: false,
		},
		{
			expression:  `checksum(input.missing_key)`, // Should throw an error due to missing key
			input:       cel.NewWorkflowStringInput(),
			expected:    "",
			expectError: true,
		},
		{
			expression:  `input.custom.value + 1234`, // Invalid expression (mismatched types), expecting error
			input:       cel.NewWorkflowStringInput(),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.ParseAndEvalWorkflowString(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Did not expect error but got one")
				assert.Equal(t, tt.expected, result, "Unexpected result")
			}
		})
	}
}
