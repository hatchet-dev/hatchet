package main

import (
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

func runBulk(workflowName string, quantity int) error {
	c, err := client.New()
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	log.Printf("pushing %d workflows in bulk", quantity)

	var workflows []*client.WorkflowRun
	for i := 0; i < quantity; i++ {
		data := map[string]interface{}{
			"username": fmt.Sprintf("echo-test-%d", i),
			"user_id":  fmt.Sprintf("1234-%d", i),
		}
		workflows = append(workflows, &client.WorkflowRun{
			Name:  workflowName,
			Input: data,
			Options: []client.RunOptFunc{
				// setting a dedupe key so these shouldn't all run
				client.WithRunMetadata(map[string]interface{}{
					// "dedupe": "dedupe1",
				}),
			},
		})

	}

	outs, err := c.Admin().BulkRunWorkflow(workflows)
	if err != nil {
		panic(fmt.Errorf("error pushing event: %w", err))
	}

	for _, out := range outs {
		log.Printf("workflow run id: %v", out)
	}

	return nil

}

func runSingles(workflowName string, quantity int) error {
	c, err := client.New()
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	log.Printf("pushing %d single workflows", quantity)

	var workflows []*client.WorkflowRun
	for i := 0; i < quantity; i++ {
		data := map[string]interface{}{
			"username": fmt.Sprintf("echo-test-%d", i),
			"user_id":  fmt.Sprintf("1234-%d", i),
		}
		workflows = append(workflows, &client.WorkflowRun{
			Name:  workflowName,
			Input: data,
			Options: []client.RunOptFunc{
				client.WithRunMetadata(map[string]interface{}{
					// "dedupe": "dedupe1",
				}),
			},
		})
	}

	for _, wf := range workflows {

		go func() {
			out, err := c.Admin().RunWorkflow(wf.Name, wf.Input, wf.Options...)
			if err != nil {
				panic(fmt.Errorf("error pushing event: %w", err))
			}

			log.Printf("workflow run id: %v", out)
		}()

	}

	return nil
}
