package worker

// import (
// 	"context"
// 	"testing"
// )

// // type actionInput struct {
// // 	Message string `json:"message"`
// // }

// // type stepOneOutput struct {
// // 	Message string `json:"message"`
// // }

// // type stepTwoOutput struct {
// // 	Message string `json:"message"`
// // }

// // func TestToWorkflowJob(t *testing.T) {
// // 	testJob := WorkflowJob{
// // 		Name:        "test",
// // 		Description: "test",
// // 		Timeout:     "1m",
// // 		Steps: []WorkflowStep{
// // 			{
// // 				ActionId: "test:test",
// // 				Function: func(ctx context.Context, input *actionInput) (result *stepOneOutput, err error) {
// // 					return nil, nil
// // 				},
// // 			},
// // 			{
// // 				ActionId: "test:test",
// // 				Function: func(ctx context.Context, input *stepOneOutput) (result *stepTwoOutput, err error) {
// // 					return nil, nil
// // 				},
// // 			},
// // 		},
// // 	}

// // 	job, err := testJob.ToWorkflowJob()

// // 	if err != nil {
// // 		t.Fatalf("could not convert workflow job: %v", err)
// // 	}

// // 	t.Fatalf("%v", job)
// // }
