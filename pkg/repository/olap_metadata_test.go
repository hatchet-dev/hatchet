package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAdditionalMetadataFilters(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(nil)
		assert.Nil(t, filters)
		assert.Equal(t, 0, count)
		assert.False(t, hasDuplicates)
	})

	t.Run("empty input", func(t *testing.T) {
		raw := []string{}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		assert.Nil(t, filters)
		assert.Equal(t, 0, count)
		assert.False(t, hasDuplicates)
	})

	t.Run("single key single value", func(t *testing.T) {
		raw := []string{"test_id:A"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 1, count)
		assert.False(t, hasDuplicates)
		assert.Equal(t, "A", filters["test_id"])
	})

	t.Run("duplicate key with same value deduplicated", func(t *testing.T) {
		raw := []string{"test_id:A", "test_id:A"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 1, count)
		assert.False(t, hasDuplicates)
		assert.Equal(t, "A", filters["test_id"])
	})

	t.Run("duplicate key with distinct values preserved as slice", func(t *testing.T) {
		raw := []string{"test_id:A", "test_id:B"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 2, count)
		assert.True(t, hasDuplicates)
		assert.Equal(t, []string{"A", "B"}, filters["test_id"])
	})

	t.Run("duplicate key with three distinct values and duplicate", func(t *testing.T) {
		raw := []string{"test_id:A", "test_id:B", "test_id:C", "test_id:A"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 3, count)
		assert.True(t, hasDuplicates)
		assert.Equal(t, []string{"A", "B", "C"}, filters["test_id"])
	})

	t.Run("multiple keys with some duplicates", func(t *testing.T) {
		raw := []string{"test_id:A", "env:prod", "test_id:B", "env:staging", "tier:frontend"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 5, count)
		assert.True(t, hasDuplicates)
		assert.Equal(t, []string{"A", "B"}, filters["test_id"])
		assert.Equal(t, []string{"prod", "staging"}, filters["env"])
		assert.Equal(t, "frontend", filters["tier"])
	})

	t.Run("value containing colon", func(t *testing.T) {
		raw := []string{"url:https://example.com/api", "url:https://other.com/api"}
		filters, count, hasDuplicates := ParseAdditionalMetadataFilters(&raw)
		require.NotNil(t, filters)
		assert.Equal(t, 2, count)
		assert.True(t, hasDuplicates)
		assert.Equal(t, []string{"https://example.com/api", "https://other.com/api"}, filters["url"])
	})
}

func TestHasMultiValueMetadata(t *testing.T) {
	t.Run("single string values", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": "A",
			"env":     "prod",
		}
		assert.False(t, HasMultiValueMetadata(m))
	})

	t.Run("single element slice", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": []string{"A"},
		}
		assert.False(t, HasMultiValueMetadata(m))
	})

	t.Run("multiple element string slice", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": []string{"A", "B"},
		}
		assert.True(t, HasMultiValueMetadata(m))
	})

	t.Run("multiple element interface slice", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": []interface{}{"A", "B"},
		}
		assert.True(t, HasMultiValueMetadata(m))
	})
}

func TestAppendMetadataKeyValues(t *testing.T) {
	t.Run("single string value", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": "A",
		}
		keys, values := AppendMetadataKeyValues(m)
		assert.Equal(t, []string{"test_id"}, keys)
		assert.Equal(t, []string{"A"}, values)
	})

	t.Run("multi-value string slice", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": []string{"A", "B"},
		}
		keys, values := AppendMetadataKeyValues(m)
		assert.Equal(t, []string{"test_id", "test_id"}, keys)
		assert.Equal(t, []string{"A", "B"}, values)
	})

	t.Run("mixed single and multi-value", func(t *testing.T) {
		m := map[string]interface{}{
			"test_id": []string{"A", "B"},
			"env":     "prod",
		}
		keys, values := AppendMetadataKeyValues(m)
		require.Equal(t, 3, len(keys))
		require.Equal(t, 3, len(values))

		// Check pairs regardless of map iteration order
		pairs := make(map[string][]string)
		for i := range keys {
			pairs[keys[i]] = append(pairs[keys[i]], values[i])
		}
		assert.ElementsMatch(t, []string{"A", "B"}, pairs["test_id"])
		assert.Equal(t, []string{"prod"}, pairs["env"])
	})
}
