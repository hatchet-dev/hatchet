//go:build e2e

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/client"
)

func TestConcurrency(t *testing.T) {
	testutils.Prepare(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	events := make(chan string, 50)
	wfrIds := make(chan *client.Workflow, 50)
	c, err := client.New()

	if err != nil {
		panic("error creating client: " + err.Error())
	}
	cleanup, err := run(c, events, wfrIds)
	if err != nil {
		t.Fatalf("/run() error = %v", err)
	}

	var items []string
	var workflowRunIds []*client.WorkflowResult
	var wg sync.WaitGroup
	done := make(chan struct{})
outer:
	for {

		select {
		case item := <-events:
			items = append(items, item)
			if len(items) > 2 {
				fmt.Println("got 2 events")
				break outer
			}
		case <-ctx.Done():
			fmt.Println("context done")
			break outer

		case wfrId := <-wfrIds:
			fmt.Println("got wfr id")
			go func(workflow *client.Workflow) {
				wg.Add(1)
				defer wg.Done()
				wfr, err := workflow.Result()
				workflowRunIds = append(workflowRunIds, wfr)
				if err != nil {
					panic(fmt.Errorf("error getting workflow run result: %w", err))
				}
			}(wfrId)

		}
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {

	case <-time.After(10 * time.Second):
		fmt.Println("timeout waiting for workflow run results")
	}

	// our workflow run ids should have only one succeeded everyone else should have failed
	stateCount := make(map[string]int)

	if len(workflowRunIds) != 20 {
		t.Fatalf("expected 20 workflow run ids, got %d", len(workflowRunIds))
	}

	for _, wfrId := range workflowRunIds {
		state, err := getWorkflowStateForWorkflowRunId(c, ctx, wfrId)

		fmt.Println("state: ", state)
		if err != nil {
			t.Fatalf("error getting workflow state: %v", err)
		}
		stateCount[state]++
	}

	assert.Equal(t, 1, stateCount["SUCCEEDED"])
	assert.Equal(t, 19, stateCount["CANCELLED_BY_CONCURRENCY_LIMIT"])

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}

}

func getWorkflowStateForWorkflowRunId(client client.Client, ctx context.Context, wfr *client.WorkflowResult) (string, error) {

	stepOneOutput := &stepOneOutput{}

	err := wfr.StepOutput("step-one", stepOneOutput)
	if err != nil {

		if err.Error() == "step run failed: this step run was cancelled due to CANCELLED_BY_CONCURRENCY_LIMIT" {
			return "CANCELLED_BY_CONCURRENCY_LIMIT", nil
		}

		// this happens if we cancel before the workflow is run
		if err.Error() == "step output for step-one not found" {
			return "CANCELLED_BY_CONCURRENCY_LIMIT", nil
		}

		fmt.Println("error getting step output: %w", err)
		return "", err
	}

	return "SUCCEEDED", nil
}
