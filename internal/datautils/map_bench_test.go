package datautils

import (
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"
)

const testJSON = `{
	"correlationId": "test-correlation-123",
	"userId": "user-456",
	"requestId": "req-789",
	"timestamp": "2024-01-01T00:00:00Z",
	"metadata": {
		"nested": {
			"deep": {
				"value": "test"
			}
		}
	},
	"tags": ["tag1", "tag2", "tag3"]
}`

func BenchmarkExtractCorrelationId_Old(b *testing.B) {
	extractOld := func(additionalMetadata string) *string {
		if additionalMetadata == "" {
			return nil
		}

		var metadata map[string]any
		if err := json.Unmarshal([]byte(additionalMetadata), &metadata); err != nil {
			return nil
		}

		if corrId, exists := metadata["correlationId"]; exists {
			if corrIdStr, ok := corrId.(string); ok {
				return &corrIdStr
			}
		}

		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := extractOld(testJSON)
		if result == nil {
			b.Fatal("expected result")
		}
	}
}

func BenchmarkExtractCorrelationId_New(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := gjson.Get(testJSON, "correlationId")
		if !result.Exists() {
			b.Fatal("expected result")
		}
		val := result.String()
		_ = val
	}
}

func BenchmarkExtractCorrelationId_Empty(b *testing.B) {
	extractOld := func(additionalMetadata string) *string {
		if additionalMetadata == "" {
			return nil
		}
		var metadata map[string]any
		if err := json.Unmarshal([]byte(additionalMetadata), &metadata); err != nil {
			return nil
		}
		if corrId, exists := metadata["correlationId"]; exists {
			if corrIdStr, ok := corrId.(string); ok {
				return &corrIdStr
			}
		}
		return nil
	}

	b.Run("Old", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := extractOld("")
			_ = result
		}
	})

	b.Run("New", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if "" == "" {
				continue
			}
			result := gjson.Get("", "correlationId")
			_ = result
		}
	})
}

func BenchmarkToJSONMap_RoundTrip(b *testing.B) {
	type testStruct struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Tags        []string               `json:"tags"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	data := &testStruct{
		Name:        "test",
		Description: "test description",
		Tags:        []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": true,
		},
	}

	b.Run("Old_RoundTrip", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				b.Fatal(err)
			}

			var dataMap map[string]interface{}
			err = json.Unmarshal(jsonBytes, &dataMap)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("New_Direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
