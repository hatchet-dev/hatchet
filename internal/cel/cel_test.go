//go:build !e2e && !load && !rampup && !integration

package cel_test

import (
	"fmt"
	"testing"

	"github.com/google/cel-go/common/types"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/cel"

	"github.com/stretchr/testify/assert"
)

func TestCELParser(t *testing.T) {
	parser := cel.NewCELParser()
	dummyUuid := uuid.New()

	tests := []struct {
		expression  string
		input       cel.Input
		expected    string
		expectError bool
	}{
		{
			expression: `has(input.custom.value) ? input.custom.value : "default"`,
			input: cel.NewInput(
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
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{}),
			),
			expected:    "default",
			expectError: false,
		},
		{
			expression: `checksum(input.custom.value)`,
			input: cel.NewInput(
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
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "concatenate ",
					},
				}),
				cel.WithWorkflowRunID(dummyUuid),
			),
			expected:    fmt.Sprintf("concatenate %s", dummyUuid.String()),
			expectError: false,
		},
		{
			expression:  `checksum(input.missing_key)`, // Should throw an error due to missing key
			input:       cel.NewInput(),
			expected:    "",
			expectError: true,
		},
		{
			expression:  `input.custom.value + 1234`, // Invalid expression (mismatched types), expecting error
			input:       cel.NewInput(),
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

func TestCELParserDebugExpression(t *testing.T) {
	parser := cel.NewCELParser()
	dummyUuid := uuid.New()

	tests := []struct {
		expression  string
		input       cel.Input
		expectError bool
		expectBool  *bool
		expectStr   *string
		expectInt   *int
	}{
		// --- boolean expressions (regression: must still work) ---
		{
			expression: `input.key == 'value'`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{"key": "value"}),
			),
			expectBool: boolPtr(true),
		},
		{
			expression: `input.priority > 5`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{"priority": 10}),
			),
			expectBool: boolPtr(true),
		},
		{
			expression: `payload.tier == 'gold'`,
			input: cel.NewInput(
				cel.WithPayload(map[string]interface{}{"tier": "silver"}),
			),
			expectBool: boolPtr(false),
		},
		// --- string expressions (broken today, must pass after fix) ---
		{
			expression: `input.user_id`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{"user_id": "alice"}),
			),
			expectStr: strPtr("alice"),
		},
		{
			expression: `'singleton'`,
			input:      cel.NewInput(),
			expectStr:  strPtr("singleton"),
		},
		{
			expression: `'rl-' + workflow_run_id`,
			input: cel.NewInput(
				cel.WithWorkflowRunID(dummyUuid),
			),
			expectStr: strPtr("rl-" + dummyUuid.String()),
		},
		{
			expression: `payload.tier`,
			input: cel.NewInput(
				cel.WithPayload(map[string]interface{}{"tier": "gold"}),
			),
			expectStr: strPtr("gold"),
		},
		{
			expression: `event_key`,
			input: cel.NewInput(
				cel.WithEventKey("user:created"),
			),
			expectStr: strPtr("user:created"),
		},
		// --- int expressions (broken today, must pass after fix) ---
		{
			expression: `input.cost`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{"cost": 5}),
			),
			expectInt: intPtr(5),
		},
		{
			expression: `input.quota`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{"quota": 100}),
			),
			expectInt: intPtr(100),
		},
		// --- error cases (must still error after fix) ---
		{
			expression:  `input.missing_key`,
			input:       cel.NewInput(),
			expectError: true,
		},
		{
			expression:  `unknown_var`,
			input:       cel.NewInput(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.EvaluateDebugExpression(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			switch {
			case tt.expectBool != nil:
				assert.NotNil(t, result.Bool)
				assert.Equal(t, *tt.expectBool, *result.Bool)
			case tt.expectStr != nil:
				assert.NotNil(t, result.String)
				assert.Equal(t, *tt.expectStr, *result.String)
			case tt.expectInt != nil:
				assert.NotNil(t, result.Int)
				assert.Equal(t, *tt.expectInt, *result.Int)
			}
		})
	}
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestCELParserEventExpression(t *testing.T) {
	parser := cel.NewCELParser()

	tests := []struct {
		expression  string
		input       cel.Input
		expected    bool
		expectError bool
	}{
		{
			expression: `has(input.custom.value)`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "actual value",
					},
				}),
			),
			expected:    true,
			expectError: false,
		},
		{
			expression: `has(input.custom)`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{}),
			),
			expected:    false,
			expectError: false,
		},
		{
			expression: `input.custom.value > 314`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": 400,
					},
				}),
			),
			expected:    true,
			expectError: false,
		},
		{
			expression: `input.custom.value < 314`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": 400,
					},
				}),
			),
			expected:    false,
			expectError: false,
		},
		{
			expression:  `checksum(input.missing_key)`, // Should throw an error due to missing key
			input:       cel.NewInput(),
			expected:    false,
			expectError: true,
		},
		{
			expression:  `input.custom.value = 1234`, // Invalid expression (mismatched types), expecting error
			input:       cel.NewInput(),
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.EvaluateEventExpression(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Did not expect error but got one")
				assert.Equal(t, tt.expected, result, "Unexpected result")
			}
		})
	}
}
