//go:build !e2e && !load && !rampup && !integration

package hatchet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These offline tests dump the registration request to check what WithSlotCost puts in it.

func TestWithSlotCost_MapsToDefaultPool(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("heavy", sampleTaskFn, WithSlotCost(5))

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)
	assert.Equal(t, map[string]int32{"default": 5}, req.Tasks[0].SlotRequests)
}

func TestWithSlotCost_OmittedKeepsOneDefaultSlot(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("plain", sampleTaskFn)

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)
	assert.Equal(t, map[string]int32{"default": 1}, req.Tasks[0].SlotRequests)
}

func TestWithSlotCost_OneIsAccepted(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("one", sampleTaskFn, WithSlotCost(1))

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)
	assert.Equal(t, map[string]int32{"default": 1}, req.Tasks[0].SlotRequests)
}

func TestWithSlotCost_RejectsNonPositive(t *testing.T) {
	c := newTestClient()

	assert.Panics(t, func() {
		c.NewStandaloneTask("zero", sampleTaskFn, WithSlotCost(0))
	})
	assert.Panics(t, func() {
		c.NewStandaloneTask("negative", sampleTaskFn, WithSlotCost(-2))
	})
}

func TestWithSlotCost_DurableTaskUnchanged(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneDurableTask("durable", sampleDurableFn)

	req, _, _, _ := task.Dump()

	require.Len(t, req.Tasks, 1)
	assert.Equal(t, map[string]int32{"durable": 1}, req.Tasks[0].SlotRequests)
}
