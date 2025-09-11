//go:build !e2e && !load && !rampup && !integration

package digest_test

import (
	"testing"

	"github.com/hatchet-dev/hatchet/internal/digest"

	"github.com/stretchr/testify/assert"
)

func TestDigestValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]interface{}
	}{
		{
			name:   "Empty map",
			values: map[string]interface{}{},
		},
		{
			name: "Single key-value pair",
			values: map[string]interface{}{
				"key": "value",
			},
		},
		{
			name: "Multiple key-value pairs",
			values: map[string]interface{}{
				"a": 1,
				"b": 2,
				"c": 3,
			},
		},
		{
			name: "Nested map",
			values: map[string]interface{}{
				"parent": map[string]interface{}{
					"child1": 1,
					"child2": 2,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			digest1, err1 := digest.DigestValues(tc.values)
			assert.NoError(t, err1)

			// Shuffle the map order and calculate the digest again
			shuffledValues := shuffleMap(tc.values)
			digest2, err2 := digest.DigestValues(shuffledValues)
			assert.NoError(t, err2)

			// The digests should be the same regardless of key order
			assert.Equal(t, digest1, digest2)
		})
	}
}

// shuffleMap rearranges the map entries to simulate different iteration orders.
// this doesn't really do anything since Go map values are random anyway.
func shuffleMap(values map[string]interface{}) map[string]interface{} {
	shuffled := make(map[string]interface{})
	for k, v := range values {
		shuffled[k] = v
	}

	return shuffled
}
