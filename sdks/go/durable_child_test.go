package hatchet

import (
	"testing"

	"github.com/stretchr/testify/require"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
)

// fakeDurableParentContext implements the parts of Context that building a durable child's
// trigger request reads.
type fakeDurableParentContext struct {
	Context
	childIndex int
}

func (c *fakeDurableParentContext) CurChildIndex() int    { return c.childIndex }
func (c *fakeDurableParentContext) IncChildIndex()        { c.childIndex++ }
func (c *fakeDurableParentContext) WorkflowRunId() string { return "parent-run" }
func (c *fakeDurableParentContext) StepRunId() string     { return "parent-task" }

func durableChildOptsFor(t *testing.T, opts ...RunOptFunc) *v0Client.ChildWorkflowOpts {
	t.Helper()

	o := &runOpts{}
	for _, opt := range opts {
		opt(o)
	}

	childOpts, err := durableChildWorkflowOpts(&fakeDurableParentContext{}, "child", o)
	require.NoError(t, err)

	return childOpts
}

func TestWithCancelOnParentCancelSetsTriggerRequestField(t *testing.T) {
	childOpts := durableChildOptsFor(t, WithCancelOnParentCancel())

	req, err := v0Client.NewChildWorkflowTriggerRequest("child", map[string]any{}, childOpts, nil)
	require.NoError(t, err)
	require.NotNil(t, req.CancelOnParentCancel)
	require.True(t, req.GetCancelOnParentCancel())
}

func TestCancelOnParentCancelIsUnsetByDefault(t *testing.T) {
	childOpts := durableChildOptsFor(t)

	req, err := v0Client.NewChildWorkflowTriggerRequest("child", map[string]any{}, childOpts, nil)
	require.NoError(t, err)
	require.Nil(t, req.CancelOnParentCancel)
}
