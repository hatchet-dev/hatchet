package repository

import (
	"encoding/json"
	"testing"
)

const testAdditionalMetadata = `{
	"correlationId": "test-correlation-123",
	"userId": "user-456",
	"requestId": "req-789",
	"environment": "production",
	"version": "1.0.0",
	"tags": ["tag1", "tag2", "tag3"],
	"metadata": {
		"nested": {
			"key": "value"
		}
	}
}`

func BenchmarkTaskMetadataUnmarshal_Old(b *testing.B) {
	strategies := []struct{ Expression string }{
		{"expression-1"},
		{"expression-2"},
		{"expression-3"},
		{"expression-4"},
		{"expression-5"},
	}

	additionalMetadataBytes := []byte(testAdditionalMetadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for range strategies {
			var additionalMeta map[string]interface{}
			if len(additionalMetadataBytes) > 0 {
				if err := json.Unmarshal(additionalMetadataBytes, &additionalMeta); err != nil {
					b.Fatal(err)
				}
			}
		}
	}
}

func BenchmarkTaskMetadataUnmarshal_New(b *testing.B) {
	strategies := []struct{ Expression string }{
		{"expression-1"},
		{"expression-2"},
		{"expression-3"},
		{"expression-4"},
		{"expression-5"},
	}

	additionalMetadataBytes := []byte(testAdditionalMetadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var additionalMeta map[string]interface{}
		if len(additionalMetadataBytes) > 0 {
			if err := json.Unmarshal(additionalMetadataBytes, &additionalMeta); err != nil {
				b.Fatal(err)
			}
		}

		for range strategies {
			_ = additionalMeta
		}
	}
}

func BenchmarkTaskMetadataUnmarshal_MultipleStrategies(b *testing.B) {
	strategyCounts := []int{1, 5, 10, 20}

	for _, count := range strategyCounts {
		strategies := make([]struct{ Expression string }, count)
		for i := 0; i < count; i++ {
			strategies[i].Expression = "test-expression"
		}

		additionalMetadataBytes := []byte(testAdditionalMetadata)

		b.Run("Old_"+string(rune(count))+"_strategies", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for range strategies {
					var additionalMeta map[string]interface{}
					if len(additionalMetadataBytes) > 0 {
						if err := json.Unmarshal(additionalMetadataBytes, &additionalMeta); err != nil {
							b.Fatal(err)
						}
					}
				}
			}
		})

		b.Run("New_"+string(rune(count))+"_strategies", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				var additionalMeta map[string]interface{}
				if len(additionalMetadataBytes) > 0 {
					if err := json.Unmarshal(additionalMetadataBytes, &additionalMeta); err != nil {
						b.Fatal(err)
					}
				}

				for range strategies {
					_ = additionalMeta
				}
			}
		})
	}
}
