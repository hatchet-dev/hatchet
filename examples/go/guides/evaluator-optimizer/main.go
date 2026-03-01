package main

import (
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type GeneratorInput struct {
	Topic         string  `json:"topic"`
	Audience      string  `json:"audience"`
	PreviousDraft *string `json:"previous_draft,omitempty"`
	Feedback      *string `json:"feedback,omitempty"`
}

type EvaluatorInput struct {
	Draft    string `json:"draft"`
	Topic    string `json:"topic"`
	Audience string `json:"audience"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// > Step 01 Define Tasks
	generatorTask := client.NewStandaloneTask("generate-draft", func(ctx hatchet.Context, input GeneratorInput) (map[string]interface{}, error) {
		var prompt string
		if input.Feedback != nil {
			prompt = fmt.Sprintf("Improve this draft.\n\nDraft: %s\nFeedback: %s", *input.PreviousDraft, *input.Feedback)
		} else {
			prompt = fmt.Sprintf("Write a social media post about \"%s\" for %s. Under 100 words.", input.Topic, input.Audience)
		}
		return map[string]interface{}{"draft": MockGenerate(prompt)}, nil
	})

	evaluatorTask := client.NewStandaloneTask("evaluate-draft", func(ctx hatchet.Context, input EvaluatorInput) (map[string]interface{}, error) {
		result := MockEvaluate(input.Draft)
		return map[string]interface{}{"score": result.Score, "feedback": result.Feedback}, nil
	})

	// > Step 02 Optimization Loop
	optimizerTask := client.NewStandaloneDurableTask("evaluator-optimizer", func(ctx hatchet.DurableContext, input map[string]interface{}) (map[string]interface{}, error) {
		maxIterations := 3
		threshold := 0.8
		draft := ""
		feedback := ""
		topic := input["topic"].(string)
		audience := input["audience"].(string)

		for i := 0; i < maxIterations; i++ {
			genInput := GeneratorInput{Topic: topic, Audience: audience}
			if draft != "" {
				genInput.PreviousDraft = &draft
			}
			if feedback != "" {
				genInput.Feedback = &feedback
			}
			genResult, err := generatorTask.Run(ctx, genInput)
			if err != nil {
				return nil, err
			}
			draft = genResult["draft"].(string)

			evalResult, err := evaluatorTask.Run(ctx, EvaluatorInput{Draft: draft, Topic: topic, Audience: audience})
			if err != nil {
				return nil, err
			}

			score := evalResult["score"].(float64)
			if score >= threshold {
				return map[string]interface{}{"draft": draft, "iterations": i + 1, "score": score}, nil
			}
			feedback = evalResult["feedback"].(string)
		}

		return map[string]interface{}{"draft": draft, "iterations": maxIterations, "score": -1}, nil
	})

	// > Step 03 Run Worker
	worker, err := client.NewWorker("evaluator-optimizer-worker",
		hatchet.WithWorkflows(generatorTask, evaluatorTask, optimizerTask),
		hatchet.WithSlots(5),
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
