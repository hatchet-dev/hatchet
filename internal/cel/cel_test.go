//go:build !e2e && !load && !rampup && !integration

package cel_test

import (
	"crypto/sha256"
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

func TestCELParserStepRun(t *testing.T) {
	parser := cel.NewCELParser()
	dummyUuid := uuid.New()

	tests := []struct {
		expression     string
		input          cel.Input
		expectedString *string
		expectedInt    *int
		expectError    bool
	}{
		{
			expression: `parents["step1"]["result"]`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{}),
				cel.WithParents(map[string]map[string]interface{}{
					"step1": {
						"result": "parent output",
					},
				}),
			),
			expectedString: strPtr("parent output"),
			expectError:    false,
		},
		{
			expression: `has(input.group) ? input.group : parents["step1"]["group"]`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{}),
				cel.WithParents(map[string]map[string]interface{}{
					"step1": {
						"group": "from parent",
					},
				}),
			),
			expectedString: strPtr("from parent"),
			expectError:    false,
		},
		{
			// expressions which are valid in the workflow env should still work through
			// the step run path
			expression: `input.custom.value + workflow_run_id`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "concatenate ",
					},
				}),
				cel.WithWorkflowRunID(dummyUuid),
			),
			expectedString: strPtr(fmt.Sprintf("concatenate %s", dummyUuid.String())),
			expectError:    false,
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
			expectedString: strPtr("97e9269cd0514f864e6be9157998464c94776ebc7f669b449f581abdad4035f5"), // Precomputed checksum
			expectError:    false,
		},
		{
			expression: `int(parents["step1"]["units"]) * 2`,
			input: cel.NewInput(
				cel.WithParents(map[string]map[string]interface{}{
					"step1": {
						"units": 3,
					},
				}),
			),
			expectedInt: intPtr(6),
			expectError: false,
		},
		{
			expression:  `parents["missing"]["value"]`, // Should throw an error due to missing parent
			input:       cel.NewInput(cel.WithParents(map[string]map[string]interface{}{})),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.ParseAndEvalStepRun(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
				return
			}

			assert.NoError(t, err, "Did not expect error but got one")

			if tt.expectedString != nil {
				assert.NotNil(t, result.String, "Expected string output")
				assert.Equal(t, *tt.expectedString, *result.String, "Unexpected result")
			}

			if tt.expectedInt != nil {
				assert.NotNil(t, result.Int, "Expected int output")
				assert.Equal(t, *tt.expectedInt, *result.Int, "Unexpected result")
			}
		})
	}
}

func TestCELParserIdempotencyKey(t *testing.T) {
	parser := cel.NewCELParser()

	tests := []struct {
		expression  string
		input       cel.Input
		expected    string
		expectError bool
		errContains string
	}{
		{
			expression: `input.order_id`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"order_id": "order-123",
				}),
			),
			expected:    "order-123",
			expectError: false,
		},
		{
			expression: `checksum(input.order_id) + additional_metadata.region`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"order_id": "order-123",
				}),
				cel.WithAdditionalMetadata(map[string]interface{}{
					"region": "us-east-1",
				}),
			),
			expected:    fmt.Sprintf("%x", sha256.Sum256([]byte("order-123"))) + "us-east-1",
			expectError: false,
		},
		{
			// idempotency keys are evaluated before a run exists, so workflow_run_id
			// should be rejected at compile time
			expression:  `workflow_run_id`,
			input:       cel.NewInput(),
			expectError: true,
			errContains: "workflow_run_id",
		},
		{
			expression:  `input.order_id + workflow_run_id`,
			input:       cel.NewInput(cel.WithInput(map[string]interface{}{"order_id": "order-123"})),
			expectError: true,
			errContains: "workflow_run_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.ParseAndEvalIdempotencyKey(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains, "Expected error to reference the offending variable")
				}
			} else {
				assert.NoError(t, err, "Did not expect error but got one")
				assert.Equal(t, tt.expected, result, "Unexpected result")
			}
		})
	}
}

func TestCELParserIncomingWebhookExpression(t *testing.T) {
	parser := cel.NewCELParser()

	tests := []struct {
		expression  string
		input       cel.Input
		expected    string
		expectError bool
	}{
		{
			expression: `input.id`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"id": "event-123",
				}),
			),
			expected:    "event-123",
			expectError: false,
		},
		{
			expression: `checksum(headers["x-signature"])`,
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{}),
				cel.WithHeaders(map[string]string{
					"x-signature": "checksum this",
				}),
			),
			expected:    "97e9269cd0514f864e6be9157998464c94776ebc7f669b449f581abdad4035f5", // Precomputed checksum
			expectError: false,
		},
		{
			expression:  `checksum(input.missing_key)`, // Should throw an error due to missing key
			input:       cel.NewInput(),
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			result, err := parser.EvaluateIncomingWebhookExpression(tt.expression, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Did not expect error but got one")
				assert.Equal(t, tt.expected, result, "Unexpected result")
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

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
			expression: `checksum(input.custom.value) == "97e9269cd0514f864e6be9157998464c94776ebc7f669b449f581abdad4035f5"`, // Precomputed checksum
			input: cel.NewInput(
				cel.WithInput(map[string]interface{}{
					"custom": map[string]interface{}{
						"value": "checksum this",
					},
				}),
			),
			expected:    true,
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
