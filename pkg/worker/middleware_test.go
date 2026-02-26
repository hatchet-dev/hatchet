//go:build !e2e && !load && !rampup && !integration

package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
)

type testHatchetContext struct {
	context.Context
}

func (c *testHatchetContext) SetContext(ctx context.Context) {
	c.Context = ctx
}

func (c *testHatchetContext) GetContext() context.Context {
	return c.Context
}

func (c *testHatchetContext) StepOutput(step string, target interface{}) error {
	return nil
}

func (c *testHatchetContext) TriggerDataKeys() []string {
	return nil
}

func (c *testHatchetContext) TriggerData(key string, target interface{}) error {
	return nil
}

func (c *testHatchetContext) ParentOutput(task create.NamedTask, target interface{}) error {
	return nil
}

func (c *testHatchetContext) WasSkipped(task create.NamedTask) bool {
	return false
}

func (c *testHatchetContext) TriggeredByEvent() bool {
	return false
}

func (c *testHatchetContext) WorkflowInput(target interface{}) error {
	return nil
}

func (c *testHatchetContext) UserData(target interface{}) error {
	return nil
}

func (c *testHatchetContext) StepRunErrors() map[string]string {
	return make(map[string]string)
}

func (c *testHatchetContext) AdditionalMetadata() map[string]string {
	return nil
}

func (c *testHatchetContext) SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.Workflow, error) {
	panic("not implemented")
}
func (c *testHatchetContext) SpawnWorkflows(opts []*SpawnWorkflowsOpts) ([]*client.Workflow, error) {
	panic("not implemented")
}

func (c *testHatchetContext) StepName() string {
	panic("not implemented")
}

func (c *testHatchetContext) StepRunId() string {
	panic("not implemented")
}

func (c *testHatchetContext) StepId() string {
	panic("not implemented")
}

func (c *testHatchetContext) WorkflowRunId() string {
	panic("not implemented")
}

func (c *testHatchetContext) WorkflowId() *string {
	panic("not implemented")
}

func (c *testHatchetContext) WorkflowVersionId() *string {
	panic("not implemented")
}

func (c *testHatchetContext) Log(message string) {
	panic("not implemented")
}

func (c *testHatchetContext) ReleaseSlot() error {
	panic("not implemented")
}

func (c *testHatchetContext) RefreshTimeout(incrementIntervalBy string) error {
	panic("not implemented")
}

func (c *testHatchetContext) StreamEvent(message []byte) {
	panic("not implemented")
}

func (c *testHatchetContext) PutStream(message string) {
	panic("not implemented")
}

func (c *testHatchetContext) RetryCount() int {
	panic("not implemented")
}

func (c *testHatchetContext) Priority() int32 {
	panic("not implemented")
}

func (c *testHatchetContext) action() *client.Action {
	panic("not implemented")
}

func (c *testHatchetContext) CurChildIndex() int {
	panic("not implemented")
}

func (c *testHatchetContext) IncChildIndex() {
	panic("not implemented")
}

func (c *testHatchetContext) client() client.Client {
	panic("not implemented")
}

func (c *testHatchetContext) Worker() HatchetWorkerContext {
	panic("not implemented")
}

func (c *testHatchetContext) FilterPayload() map[string]interface{} {
	panic("not implemented")
}

func TestAddMiddleware(t *testing.T) {
	m := middlewares{}
	middlewareFunc := func(ctx HatchetContext, next func(HatchetContext) error) error {
		return nil
	}
	m.add(middlewareFunc)

	if len(m.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(m.middlewares))
	}
}

func TestRunAllWithNoMiddleware(t *testing.T) {
	m := middlewares{}
	err := m.runAll(&testHatchetContext{context.Background()}, func(ctx HatchetContext) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunAllWithMiddleware(t *testing.T) {
	m := middlewares{}
	called := false
	middlewareFunc := func(ctx HatchetContext, next func(HatchetContext) error) error {
		called = true
		return next(ctx)
	}
	m.add(middlewareFunc)

	err := m.runAll(&testHatchetContext{context.Background()}, func(ctx HatchetContext) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Errorf("Expected middleware to be called")
	}
}

func TestRunAllWithPropagatedContext(t *testing.T) {
	m := middlewares{}
	key := "key"
	value := "value"

	// Middleware that sets a value in the context
	middlewareFunc := func(ctx HatchetContext, next func(HatchetContext) error) error {
		ctx.SetContext(context.WithValue(ctx, key, value))

		return next(ctx)
	}
	m.add(middlewareFunc)

	// Next function that checks for the value in the context
	err := m.runAll(&testHatchetContext{context.Background()}, func(ctx HatchetContext) error {
		if ctx.Value(key) != value {
			t.Errorf("Expected value %v in context, got %v", value, ctx.Value(key))
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunAllWithErrorInMiddleware(t *testing.T) {
	m := middlewares{}
	expectedErr := errors.New("middleware error")
	middlewareFunc := func(ctx HatchetContext, next func(HatchetContext) error) error {
		return expectedErr
	}
	m.add(middlewareFunc)

	err := m.runAll(&testHatchetContext{context.Background()}, func(ctx HatchetContext) error {
		return nil
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRunAllWithErrorInNext(t *testing.T) {
	m := middlewares{}
	expectedErr := errors.New("next error")
	middlewareFunc := func(ctx HatchetContext, next func(HatchetContext) error) error {
		return next(ctx)
	}
	m.add(middlewareFunc)

	err := m.runAll(&testHatchetContext{context.Background()}, func(ctx HatchetContext) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}
