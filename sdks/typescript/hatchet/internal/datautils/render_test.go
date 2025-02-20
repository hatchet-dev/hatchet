package datautils

import (
	"encoding/json"
	"testing"
)

func TestRenderTemplateFields(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		input    map[string]interface{}
		expected map[string]interface{}
		wantErr  bool
	}{
		{
			name: "simple string template",
			data: map[string]interface{}{"testing": "datavalue"},
			input: map[string]interface{}{
				"render": "{{ .testing }}",
			},
			expected: map[string]interface{}{
				"render": "datavalue",
			},
			wantErr: false,
		},
		{
			name: "nested map template",
			data: map[string]interface{}{"testing": "nestedvalue"},
			input: map[string]interface{}{
				"nested": map[string]interface{}{
					"render": "{{ .testing }}",
				},
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"render": "nestedvalue",
				},
			},
			wantErr: false,
		},
		{
			name: "object template",
			data: map[string]interface{}{"testing": `{ "nested": "nestedvalue" }`},
			input: map[string]interface{}{
				"nested": map[string]interface{}{
					"render": "{{ .testing }}",
				},
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{
					"render": map[string]interface{}{
						"nested": "nestedvalue",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "replace object",
			data: map[string]interface{}{"testing": `{ "nested": "nestedvalue" }`},
			input: map[string]interface{}{
				"object": "{{ .testing }}",
			},
			expected: map[string]interface{}{
				"nested": "nestedvalue",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := RenderTemplateFields(tt.data, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderTemplateFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				jsonExpected, _ := json.Marshal(tt.expected)
				jsonResult, _ := json.Marshal(output)
				if string(jsonExpected) != string(jsonResult) {
					t.Errorf("Expected %v, got %v", string(jsonExpected), string(jsonResult))
				}
			}
		})
	}
}
