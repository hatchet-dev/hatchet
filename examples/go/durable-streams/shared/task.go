package shared

import (
	"encoding/json"
	"errors"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type ProgressInput struct {
	Topic string `json:"topic"`
}

type ProgressOutput struct {
	Steps int `json:"steps"`
}

type ProgressMessage struct {
	Step  int `json:"step"`
	Total int `json:"total"`
}

func ProgressTask(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Publishing
	return client.NewStandaloneTask("durable-streams-progress", func(ctx hatchet.Context, input ProgressInput) (*ProgressOutput, error) {
		total := 5

		for step := 1; step <= total; step++ {
			time.Sleep(500 * time.Millisecond)

			message, err := json.Marshal(ProgressMessage{Step: step, Total: total})
			if err != nil {
				return nil, err
			}

			if err := client.Streams().Publish(ctx, input.Topic, message); err != nil {
				return nil, err
			}
		}

		return &ProgressOutput{Steps: total}, nil
	})
}

type RetryingProgressOutput struct {
	Chunks int `json:"chunks"`
}

type RetryingProgressMessage struct {
	Attempt *int `json:"attempt,omitempty"`
	Chunk   *int `json:"chunk,omitempty"`
	Done    bool `json:"done,omitempty"`
}

func RetryingProgressTask(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Reset on retry
	return client.NewStandaloneTask("durable-streams-retrying-progress", func(ctx hatchet.Context, input ProgressInput) (*RetryingProgressOutput, error) {
		if err := client.Streams().Reset(ctx, input.Topic); err != nil {
			return nil, err
		}

		for chunk := 0; chunk < 3; chunk++ {
			attempt := ctx.RetryCount()

			message, err := json.Marshal(RetryingProgressMessage{Attempt: &attempt, Chunk: &chunk})
			if err != nil {
				return nil, err
			}

			if err := client.Streams().Publish(ctx, input.Topic, message); err != nil {
				return nil, err
			}
		}

		if ctx.RetryCount() == 0 {
			return nil, errors.New("intentional failure before the final message")
		}

		message, err := json.Marshal(RetryingProgressMessage{Done: true})
		if err != nil {
			return nil, err
		}

		if err := client.Streams().Publish(ctx, input.Topic, message); err != nil {
			return nil, err
		}

		return &RetryingProgressOutput{Chunks: 3}, nil
	}, hatchet.WithRetries(1))
}
