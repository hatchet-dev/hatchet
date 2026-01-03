package hatchet

import (
	"context"
	"reflect"
	"testing"
	"time"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/sdks/go/internal"
	"github.com/hatchet-dev/hatchet/sdks/go/internal/task"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name   string   `json:"name"`
	Age    int      `json:"age"`
	Active bool     `json:"active"`
	Height *float64 `json:"height,omitempty"`
}

type NestedStruct struct {
	ID       int                    `json:"id"`
	User     TestStruct             `json:"user"`
	Tags     []string               `json:"tags"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ComplexStruct struct {
	Timestamp time.Time         `json:"timestamp"`
	Values    []int             `json:"values"`
	Config    map[string]string `json:"config"`
	Nested    []TestStruct      `json:"nested"`
	Optional  *NestedStruct     `json:"optional,omitempty"`
}

func TestConvertInputToType_NilInput(t *testing.T) {
	expectedType := reflect.TypeOf(TestStruct{})
	result := convertInputToType(nil, expectedType)

	assert.Equal(t, reflect.Zero(expectedType), result)
	assert.Equal(t, TestStruct{}, result.Interface())
}

func TestConvertInputToType_DirectAssignable(t *testing.T) {
	input := TestStruct{Name: "John", Age: 30, Active: true}
	expectedType := reflect.TypeOf(TestStruct{})

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_MapToStruct(t *testing.T) {
	input := map[string]interface{}{
		"name":   "Alice",
		"age":    25,
		"active": true,
	}
	expectedType := reflect.TypeOf(TestStruct{})

	result := convertInputToType(input, expectedType)

	expected := TestStruct{Name: "Alice", Age: 25, Active: true}
	assert.Equal(t, expected, result.Interface())
}

func TestConvertInputToType_MapToStructWithPointer(t *testing.T) {
	height := 5.8
	input := map[string]interface{}{
		"name":   "Bob",
		"age":    35,
		"active": false,
		"height": height,
	}
	expectedType := reflect.TypeOf(TestStruct{})

	result := convertInputToType(input, expectedType)

	expected := TestStruct{Name: "Bob", Age: 35, Active: false, Height: &height}
	assert.Equal(t, expected, result.Interface())
}

func TestConvertInputToType_NestedStruct(t *testing.T) {
	input := map[string]interface{}{
		"id": 123,
		"user": map[string]interface{}{
			"name":   "Charlie",
			"age":    28,
			"active": true,
		},
		"tags": []interface{}{"admin", "user"},
		"metadata": map[string]interface{}{
			"role":   "admin",
			"region": "us-west",
		},
	}
	expectedType := reflect.TypeOf(NestedStruct{})

	result := convertInputToType(input, expectedType)

	expected := NestedStruct{
		ID:       123,
		User:     TestStruct{Name: "Charlie", Age: 28, Active: true},
		Tags:     []string{"admin", "user"},
		Metadata: map[string]interface{}{"role": "admin", "region": "us-west"},
	}
	assert.Equal(t, expected, result.Interface())
}

func TestConvertInputToType_ComplexStruct(t *testing.T) {
	timestamp := "2023-10-15T10:30:00Z"
	input := map[string]interface{}{
		"timestamp": timestamp,
		"values":    []interface{}{1, 2, 3, 4, 5},
		"config": map[string]interface{}{
			"env":     "production",
			"version": "1.0.0",
		},
		"nested": []interface{}{
			map[string]interface{}{
				"name":   "User1",
				"age":    20,
				"active": true,
			},
			map[string]interface{}{
				"name":   "User2",
				"age":    30,
				"active": false,
			},
		},
	}
	expectedType := reflect.TypeOf(ComplexStruct{})

	result := convertInputToType(input, expectedType)
	resultStruct := result.Interface().(ComplexStruct)

	assert.Equal(t, []int{1, 2, 3, 4, 5}, resultStruct.Values)
	assert.Equal(t, map[string]string{"env": "production", "version": "1.0.0"}, resultStruct.Config)
	assert.Len(t, resultStruct.Nested, 2)
	assert.Equal(t, "User1", resultStruct.Nested[0].Name)
	assert.Equal(t, 20, resultStruct.Nested[0].Age)
	assert.Equal(t, "User2", resultStruct.Nested[1].Name)
	assert.Equal(t, 30, resultStruct.Nested[1].Age)
}

func TestConvertInputToType_EmptyStruct(t *testing.T) {
	input := map[string]interface{}{}
	expectedType := reflect.TypeOf(TestStruct{})

	result := convertInputToType(input, expectedType)

	expected := TestStruct{}
	assert.Equal(t, expected, result.Interface())
}

func TestConvertInputToType_InvalidJSONMarshaling(t *testing.T) {
	input := make(chan int)
	expectedType := reflect.TypeOf(TestStruct{})

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_MalformedMapData(t *testing.T) {
	input := map[string]interface{}{
		"name": 123,
		"age":  "not-a-number",
	}
	expectedType := reflect.TypeOf(TestStruct{})

	defer func() {
		if r := recover(); r != nil {
			assert.NotNil(t, r)
		} else {
			t.Fatal("Expected panic but none occurred")
		}
	}()

	_ = convertInputToType(input, expectedType)
}

func TestConvertInputToType_NonStructType(t *testing.T) {
	input := map[string]interface{}{"key": "value"}
	expectedType := reflect.TypeOf("")

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_StringToString(t *testing.T) {
	input := "hello world"
	expectedType := reflect.TypeOf("")

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_IntToInt(t *testing.T) {
	input := 42
	expectedType := reflect.TypeOf(0)

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_SliceToSlice(t *testing.T) {
	input := []string{"a", "b", "c"}
	expectedType := reflect.TypeOf([]string{})

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_MapToMap(t *testing.T) {
	input := map[string]int{"one": 1, "two": 2}
	expectedType := reflect.TypeOf(map[string]int{})

	result := convertInputToType(input, expectedType)

	assert.Equal(t, input, result.Interface())
}

func TestConvertInputToType_PointerStruct(t *testing.T) {
	input := map[string]interface{}{
		"id": 456,
		"user": map[string]interface{}{
			"name": "Diana",
			"age":  32,
		},
		"tags": []interface{}{"vip"},
		"metadata": map[string]interface{}{
			"premium": "true",
		},
	}
	expectedType := reflect.TypeOf(NestedStruct{})

	result := convertInputToType(input, expectedType)

	expected := NestedStruct{
		ID:       456,
		User:     TestStruct{Name: "Diana", Age: 32},
		Tags:     []string{"vip"},
		Metadata: map[string]interface{}{"premium": "true"},
	}
	assert.Equal(t, expected, result.Interface())
}

func TestConvertInputToType_StructWithNilPointer(t *testing.T) {
	input := map[string]interface{}{
		"timestamp": "2023-01-01T00:00:00Z",
		"values":    []interface{}{1, 2, 3},
		"config":    map[string]interface{}{"key": "value"},
		"nested":    []interface{}{},
		"optional":  nil,
	}
	expectedType := reflect.TypeOf(ComplexStruct{})

	result := convertInputToType(input, expectedType)
	resultStruct := result.Interface().(ComplexStruct)

	assert.Nil(t, resultStruct.Optional)
	assert.Equal(t, []int{1, 2, 3}, resultStruct.Values)
	assert.Equal(t, map[string]string{"key": "value"}, resultStruct.Config)
	assert.Empty(t, resultStruct.Nested)
}

func TestConvertInputToType_NumberTypesConversion(t *testing.T) {
	input := map[string]interface{}{
		"name":   "Eve",
		"age":    float64(27),
		"active": 1,
	}
	expectedType := reflect.TypeOf(TestStruct{})

	defer func() {
		if r := recover(); r != nil {
			assert.NotNil(t, r)
		} else {
			t.Fatal("Expected panic but none occurred")
		}
	}()

	_ = convertInputToType(input, expectedType)
}

type structIn struct{ V string }
type structOut struct{ V int }

type mockDecl struct{}

func (m *mockDecl) Name() string { return "test" }

func (m *mockDecl) Task(opts create.WorkflowTask[any, any], fn func(ctx worker.HatchetContext, input any) (interface{}, error)) *task.TaskDeclaration[any] {
	return &task.TaskDeclaration[any]{Name: opts.Name}
}

func (m *mockDecl) DurableTask(opts create.WorkflowTask[any, any], fn func(ctx worker.DurableHatchetContext, input any) (interface{}, error)) *task.DurableTaskDeclaration[any] {
	return &task.DurableTaskDeclaration[any]{Name: opts.Name}
}

func (m *mockDecl) OnFailure(opts create.WorkflowOnFailureTask[any, any], fn func(ctx worker.HatchetContext, input any) (interface{}, error)) *task.OnFailureTaskDeclaration[any] {
	return &task.OnFailureTaskDeclaration[any]{}
}

func (m *mockDecl) Cron(ctx context.Context, name string, cronExpr string, input any, opts ...v0Client.RunOptFunc) (*rest.CronWorkflows, error) {
	return nil, nil
}

func (m *mockDecl) Schedule(ctx context.Context, triggerAt time.Time, input any, opts ...v0Client.RunOptFunc) (*rest.ScheduledWorkflows, error) {
	return nil, nil
}

func (m *mockDecl) Get(ctx context.Context) (*rest.Workflow, error) { return nil, nil }

func (m *mockDecl) Metrics(ctx context.Context, opts ...rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error) {
	return nil, nil
}

func (m *mockDecl) QueueMetrics(ctx context.Context, opts ...rest.TenantGetQueueMetricsParams) (*rest.TenantQueueMetrics, error) {
	return nil, nil
}

func (m *mockDecl) Dump() (*contracts.CreateWorkflowVersionRequest, []internal.NamedFunction, []internal.NamedFunction, internal.WrappedTaskFn) {
	return nil, nil, nil, nil
}

func TestNewTask_AcceptsStructTypes_NoPanic(t *testing.T) {
	w := &Workflow{}
	// inject mock declaration to avoid nil deref later
	w.declaration = &mockDecl{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("did not expect panic for struct types, got: %v", r)
		}
	}()

	_ = w.NewTask("ok", func(ctx Context, in structIn) (structOut, error) {
		return structOut{V: 1}, nil
	})
}

func TestNewDurableTask_AcceptsStructTypes_NoPanic(t *testing.T) {
	w := &Workflow{}
	w.declaration = &mockDecl{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("did not expect panic for struct types (durable), got: %v", r)
		}
	}()

	_ = w.NewDurableTask("ok-durable", func(ctx DurableContext, in structIn) (structOut, error) {
		return structOut{V: 2}, nil
	})
}

func TestOnFailure_AcceptsStructTypes_NoPanic(t *testing.T) {
	w := &Workflow{}
	w.declaration = &mockDecl{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("did not expect panic for struct types in OnFailure, got: %v", r)
		}
	}()

	w.OnFailure(func(ctx Context, in structIn) (structOut, error) {
		return structOut{V: 3}, nil
	})
}

func TestNewTask_RejectsPrimitiveTypes(t *testing.T) {
	w := &Workflow{}

	// primitive input
	assert.Panics(t, func() {
		w.NewTask("bad-in", func(ctx Context, in int) (structOut, error) { return structOut{}, nil })
	})

	// primitive output
	assert.Panics(t, func() {
		w.NewTask("bad-out", func(ctx Context, in structIn) (int, error) { return 0, nil })
	})
}
