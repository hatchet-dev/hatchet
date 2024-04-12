package worker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func namedFunction() {}

func TestGetFnNameAnon(t *testing.T) {
	fn := func() {}

	name := getFnName(fn)

	if name != "TestGetFnNameAnon-func1" {
		t.Fatalf("expected function name to be TestGetFnNameAnon-func1, got %s", name)
	}

	name = getFnName(func() {})

	if name != "TestGetFnNameAnon-func2" {
		t.Fatalf("expected function name to be TestGetFnNameAnon-func2, got %s", name)
	}

	name = getFnName(namedFunction)

	if name != "namedFunction" {
		t.Fatalf("expected function name to be namedFunction, got %s", name)
	}
}

type actionInput struct {
	Message string `json:"message"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

type stepTwoOutput struct {
	Message string `json:"message"`
}

func TestToWorkflowJob(t *testing.T) {
	testJob := WorkflowJob{
		Name:        "test",
		Description: "test",
		Steps: []*WorkflowStep{
			{
				Function: func(ctx context.Context, input *actionInput) (result *stepOneOutput, err error) {
					return nil, nil
				},
			},
			{
				Function: func(ctx context.Context, input *stepOneOutput) (result *stepTwoOutput, err error) {
					return nil, nil
				},
			},
		},
	}

	workflow := testJob.ToWorkflow("default", "")

	assert.Equal(t, "test", workflow.Name)
}

func TestFnToWorkflow(t *testing.T) {
	workflow := Fn(func(ctx context.Context, input *actionInput) (result *stepOneOutput, err error) {
		return nil, nil
	}).ToWorkflow("default", "")

	assert.Equal(t, "TestFnToWorkflow-func1", workflow.Name)
}
