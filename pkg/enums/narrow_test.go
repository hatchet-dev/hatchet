package enums_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/enums"
)

func TestNarrow(t *testing.T) {
	got, ok := enums.Narrow[enums.V1TaskStatus](enums.V1ReadableStatusOlapRUNNING)
	assert.True(t, ok)
	assert.Equal(t, enums.V1TaskStatusRUNNING, got)

	_, ok = enums.Narrow[enums.V1TaskStatus](enums.V1ReadableStatusOlapEVICTED)
	assert.False(t, ok)
}
