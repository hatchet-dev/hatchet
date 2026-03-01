package main

import (
	"log"
	"sync"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type MessageInput struct {
	Message string `json:"message"`
}

type ContentInput struct {
	Content string `json:"content"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Parallel Tasks
	contentTask := client.NewStandaloneTask("generate-content", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		return map[string]interface{}{"content": MockGenerateContent(input.Message)}, nil
	})

	safetyTask := client.NewStandaloneTask("safety-check", func(ctx hatchet.Context, input MessageInput) (map[string]interface{}, error) {
		result := MockSafetyCheck(input.Message)
		return map[string]interface{}{"safe": result.Safe, "reason": result.Reason}, nil
	})

	evaluateTask := client.NewStandaloneTask("evaluate-content", func(ctx hatchet.Context, input ContentInput) (map[string]interface{}, error) {
		result := MockEvaluateContent(input.Content)
		return map[string]interface{}{"score": result.Score, "approved": result.Approved}, nil
	})

	// > Step 02 Sectioning
	sectioningTask := client.NewStandaloneDurableTask("parallel-sectioning", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		msg := input["message"].(string)

		var contentTr, safetyTr *hatchet.TaskResult
		var contentErr, safetyErr error
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			contentTr, contentErr = contentTask.Run(ctx, MessageInput{Message: msg})
		}()
		go func() {
			defer wg.Done()
			safetyTr, safetyErr = safetyTask.Run(ctx, MessageInput{Message: msg})
		}()
		wg.Wait()

		if contentErr != nil {
			return nil, contentErr
		}
		if safetyErr != nil {
			return nil, safetyErr
		}
		var contentResult, safetyResult map[string]interface{}
		if err := contentTr.Into(&contentResult); err != nil {
			return nil, err
		}
		if err := safetyTr.Into(&safetyResult); err != nil {
			return nil, err
		}

		if safe, ok := safetyResult["safe"].(bool); !ok || !safe {
			return map[string]interface{}{"blocked": true, "reason": safetyResult["reason"]}, nil
		}
		return map[string]interface{}{"blocked": false, "content": contentResult["content"]}, nil
	})

	// > Step 03 Voting
	votingTask := client.NewStandaloneDurableTask("parallel-voting", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		content := input["content"].(string)
		numVoters := 3
		taskResults := make([]*hatchet.TaskResult, numVoters)
		errs := make([]error, numVoters)

		var wg sync.WaitGroup
		for i := 0; i < numVoters; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				taskResults[idx], errs[idx] = evaluateTask.Run(ctx, ContentInput{Content: content})
			}(i)
		}
		wg.Wait()

		results := make([]map[string]interface{}, numVoters)
		for i := 0; i < numVoters; i++ {
			if errs[i] != nil {
				return nil, errs[i]
			}
			if err := taskResults[i].Into(&results[i]); err != nil {
				return nil, err
			}
		}

		approvals := 0
		totalScore := 0.0
		for _, r := range results {
			if approved, ok := r["approved"].(bool); ok && approved {
				approvals++
			}
			if score, ok := r["score"].(float64); ok {
				totalScore += score
			}
		}

		return map[string]interface{}{
			"approved":     approvals >= 2,
			"averageScore": totalScore / float64(numVoters),
			"votes":        numVoters,
		}, nil
	})

	// > Step 04 Run Worker
	worker, err := client.NewWorker("parallelization-worker",
		hatchet.WithWorkflows(contentTask, safetyTask, evaluateTask, sectioningTask, votingTask),
		hatchet.WithSlots(10),
		hatchet.WithDurableSlots(5),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
