package mergent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// ProcessImage simulates image processing
func ProcessImage(imageURL string, filters []string) (map[string]interface{}, error) {
	// Do some image processing
	return map[string]interface{}{
		"url":    imageURL,
		"size":   100,
		"format": "png",
	}, nil
}

// > Before (Mergent)
type MergentRequest struct {
	ImageURL string   `json:"image_url"`
	Filters  []string `json:"filters"`
}

type MergentResponse struct {
	Success      bool   `json:"success"`
	ProcessedURL string `json:"processed_url"`
}

func ProcessImageMergent(req MergentRequest) (*MergentResponse, error) {
	result, err := ProcessImage(req.ImageURL, req.Filters)
	if err != nil {
		return nil, err
	}

	return &MergentResponse{
		Success:      true,
		ProcessedURL: result["url"].(string),
	}, nil
}


// > After (Hatchet)
type ImageProcessInput struct {
	ImageURL string   `json:"image_url"`
	Filters  []string `json:"filters"`
}

type ImageProcessOutput struct {
	ProcessedURL string `json:"processed_url"`
	Metadata     struct {
		Size           int      `json:"size"`
		Format         string   `json:"format"`
		AppliedFilters []string `json:"applied_filters"`
	} `json:"metadata"`
}

func ImageProcessor(client *hatchet.Client) *hatchet.StandaloneTask {
	processor := client.NewStandaloneTask("image-processor",
		func(ctx hatchet.Context, input ImageProcessInput) (*ImageProcessOutput, error) {
			result, err := ProcessImage(input.ImageURL, input.Filters)
			if err != nil {
				return nil, fmt.Errorf("processing image: %w", err)
			}

			if result["url"] == "" {
				return nil, fmt.Errorf("processing failed to generate URL")
			}

			output := &ImageProcessOutput{
				ProcessedURL: result["url"].(string),
				Metadata: struct {
					Size           int      `json:"size"`
					Format         string   `json:"format"`
					AppliedFilters []string `json:"applied_filters"`
				}{
					Size:           result["size"].(int),
					Format:         result["format"].(string),
					AppliedFilters: input.Filters,
				},
			}

			return output, nil
		},
	)

	// Example of running a task
	_ = func() error {
		// > Running a task
		result, err := processor.Run(context.Background(), ImageProcessInput{
			ImageURL: "https://example.com/image.png",
			Filters:  []string{"blur"},
		})
		if err != nil {
			return err
		}

		fmt.Printf("Result: %+v\n", result)

		return nil
	}

	// Example of registering a task on a worker
	_ = func() error {
		// > Declaring a Worker
		w, err := client.NewWorker("image-processor-worker", hatchet.WithWorkflows(processor))
		if err != nil {
			return err
		}

		err = w.StartBlocking(context.Background())
		if err != nil {
			return err
		}

		return nil
	}

	return processor
}

func RunMergentTask() error {
	return nil
}

func RunningTasks(client *hatchet.Client) error {
	// > Running a task (Mergent)
	task := struct {
		Request struct {
			URL     string            `json:"url"`
			Body    string            `json:"body"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Name  string `json:"name"`
		Queue string `json:"queue"`
	}{
		Request: struct {
			URL     string            `json:"url"`
			Body    string            `json:"body"`
			Headers map[string]string `json:"headers"`
		}{
			URL: "https://example.com",
			Headers: map[string]string{
				"Authorization": "fake-secret-token",
				"Content-Type":  "application/json",
			},
			Body: "Hello, world!",
		},
		Name:  "4cf95241-fa19-47ef-8a67-71e483747649",
		Queue: "default",
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshaling task: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.mergent.co/v2/tasks", bytes.NewBuffer(taskJSON))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer <API_KEY>")
	req.Header.Add("Content-Type", "application/json")

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer res.Body.Close()

	fmt.Printf("Mergent task created with status: %d\n", res.StatusCode)

	// > Running a task (Hatchet)
	processor := ImageProcessor(client)

	result, err := processor.Run(context.Background(), ImageProcessInput{
		ImageURL: "https://example.com/image.png",
		Filters:  []string{"blur"},
	})
	if err != nil {
		return err
	}
	fmt.Printf("Result: %+v\n", result)

	// > Scheduling tasks (Hatchet)
	// Schedule the task to run at a specific time
	scheduleRef, err := client.Schedules().Create(
		context.Background(),
		"image-processor",
		features.CreateScheduledRunTrigger{
			TriggerAt: time.Now().Add(time.Second * 10),
			Input: map[string]interface{}{
				"image_url": "https://example.com/image.png",
				"filters":   []string{"blur"},
			},
		},
	)
	if err != nil {
		return err
	}

	// or schedule to run every hour
	cronRef, err := client.Crons().Create(
		context.Background(),
		"image-processor",
		features.CreateCronTrigger{
			Name:       "run-hourly",
			Expression: "0 * * * *",
			Input: map[string]interface{}{
				"image_url": "https://example.com/image.png",
				"filters":   []string{"blur"},
			},
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("Scheduled tasks with refs: %+v, %+v\n", scheduleRef, cronRef)
	return nil
}
