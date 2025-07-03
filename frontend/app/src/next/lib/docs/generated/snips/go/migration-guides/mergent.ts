import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package migration_guides\n\nimport (\n\t"bytes"\n\t"context"\n\t"encoding/json"\n\t"fmt"\n\t"net/http"\n\t"time"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\tv1worker "github.com/hatchet-dev/hatchet/pkg/v1/worker"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\n// ProcessImage simulates image processing\nfunc ProcessImage(imageURL string, filters []string) (map[string]interface{}, error) {\n\t// Do some image processing\n\treturn map[string]interface{}{\n\t\t"url":    imageURL,\n\t\t"size":   100,\n\t\t"format": "png",\n\t}, nil\n}\n\n// > Before (Mergent)\ntype MergentRequest struct {\n\tImageURL string   `json:"image_url"`\n\tFilters  []string `json:"filters"`\n}\n\ntype MergentResponse struct {\n\tSuccess      bool   `json:"success"`\n\tProcessedURL string `json:"processed_url"`\n}\n\nfunc ProcessImageMergent(req MergentRequest) (*MergentResponse, error) {\n\tresult, err := ProcessImage(req.ImageURL, req.Filters)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn &MergentResponse{\n\t\tSuccess:      true,\n\t\tProcessedURL: result["url"].(string),\n\t}, nil\n}\n\n\n// > After (Hatchet)\ntype ImageProcessInput struct {\n\tImageURL string   `json:"image_url"`\n\tFilters  []string `json:"filters"`\n}\n\ntype ImageProcessOutput struct {\n\tProcessedURL string `json:"processed_url"`\n\tMetadata     struct {\n\t\tSize           int      `json:"size"`\n\t\tFormat         string   `json:"format"`\n\t\tAppliedFilters []string `json:"applied_filters"`\n\t} `json:"metadata"`\n}\n\nfunc ImageProcessor(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ImageProcessInput, ImageProcessOutput] {\n\tprocessor := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "image-processor",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input ImageProcessInput) (*ImageProcessOutput, error) {\n\t\t\tresult, err := ProcessImage(input.ImageURL, input.Filters)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, fmt.Errorf("processing image: %w", err)\n\t\t\t}\n\n\t\t\tif result["url"] == "" {\n\t\t\t\treturn nil, fmt.Errorf("processing failed to generate URL")\n\t\t\t}\n\n\t\t\toutput := &ImageProcessOutput{\n\t\t\t\tProcessedURL: result["url"].(string),\n\t\t\t\tMetadata: struct {\n\t\t\t\t\tSize           int      `json:"size"`\n\t\t\t\t\tFormat         string   `json:"format"`\n\t\t\t\t\tAppliedFilters []string `json:"applied_filters"`\n\t\t\t\t}{\n\t\t\t\t\tSize:           result["size"].(int),\n\t\t\t\t\tFormat:         result["format"].(string),\n\t\t\t\t\tAppliedFilters: input.Filters,\n\t\t\t\t},\n\t\t\t}\n\n\t\t\treturn output, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\t// Example of running a task\n\t_ = func() error {\n\t\t// > Running a task\n\t\tresult, err := processor.Run(context.Background(), ImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tfmt.Printf("Result: %+v\\n", result)\n\t\treturn nil\n\t}\n\n\t// Example of registering a task on a worker\n\t_ = func() error {\n\t\t// > Declaring a Worker\n\t\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\t\tName: "image-processor-worker",\n\t\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\t\tprocessor,\n\t\t\t},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\terr = w.StartBlocking(context.Background())\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\treturn nil\n\t}\n\n\treturn processor\n}\n\nfunc RunMergentTask() error {\n\n\treturn nil\n}\n\nfunc RunningTasks(hatchet v1.HatchetClient) error {\n\t// > Running a task (Mergent)\n\ttask := struct {\n\t\tRequest struct {\n\t\t\tURL     string            `json:"url"`\n\t\t\tBody    string            `json:"body"`\n\t\t\tHeaders map[string]string `json:"headers"`\n\t\t} `json:"request"`\n\t\tName  string `json:"name"`\n\t\tQueue string `json:"queue"`\n\t}{\n\t\tRequest: struct {\n\t\t\tURL     string            `json:"url"`\n\t\t\tBody    string            `json:"body"`\n\t\t\tHeaders map[string]string `json:"headers"`\n\t\t}{\n\t\t\tURL: "https://example.com",\n\t\t\tHeaders: map[string]string{\n\t\t\t\t"Authorization": "fake-secret-token",\n\t\t\t\t"Content-Type":  "application/json",\n\t\t\t},\n\t\t\tBody: "Hello, world!",\n\t\t},\n\t\tName:  "4cf95241-fa19-47ef-8a67-71e483747649",\n\t\tQueue: "default",\n\t}\n\n\ttaskJSON, err := json.Marshal(task)\n\tif err != nil {\n\t\treturn fmt.Errorf("marshaling task: %w", err)\n\t}\n\n\treq, err := http.NewRequest(http.MethodPost, "https://api.mergent.co/v2/tasks", bytes.NewBuffer(taskJSON))\n\tif err != nil {\n\t\treturn fmt.Errorf("creating request: %w", err)\n\t}\n\n\treq.Header.Add("Authorization", "Bearer <API_KEY>")\n\treq.Header.Add("Content-Type", "application/json")\n\n\tclient := &http.Client{}\n\tres, err := client.Do(req)\n\tif err != nil {\n\t\treturn fmt.Errorf("sending request: %w", err)\n\t}\n\tdefer res.Body.Close()\n\n\tfmt.Printf("Mergent task created with status: %d\\n", res.StatusCode)\n\n\t// > Running a task (Hatchet)\n\tprocessor := ImageProcessor(hatchet)\n\n\tresult, err := processor.Run(context.Background(), ImageProcessInput{\n\t\tImageURL: "https://example.com/image.png",\n\t\tFilters:  []string{"blur"},\n\t})\n\tif err != nil {\n\t\treturn err\n\t}\n\tfmt.Printf("Result: %+v\\n", result)\n\n\t// > Scheduling tasks (Hatchet)\n\t// Schedule the task to run at a specific time\n\tscheduleRef, err := processor.Schedule(\n\t\tcontext.Background(),\n\t\ttime.Now().Add(time.Second*10),\n\t\tImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\t// or schedule to run every hour\n\tcronRef, err := processor.Cron(\n\t\tcontext.Background(),\n\t\t"run-hourly",\n\t\t"0 * * * *",\n\t\tImageProcessInput{\n\t\t\tImageURL: "https://example.com/image.png",\n\t\t\tFilters:  []string{"blur"},\n\t\t},\n\t)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\tfmt.Printf("Scheduled tasks with refs: %+v, %+v\\n", scheduleRef, cronRef)\n\treturn nil\n}\n',
  source: 'out/go/migration-guides/mergent.go',
  blocks: {
    before_mergent: {
      start: 30,
      stop: 51,
    },
    after_hatchet: {
      start: 54,
      stop: 99,
    },
    running_a_task: {
      start: 104,
      stop: 112,
    },
    declaring_a_worker: {
      start: 118,
      stop: 131,
    },
    running_a_task_mergent: {
      start: 144,
      stop: 189,
    },
    running_a_task_hatchet: {
      start: 192,
      stop: 201,
    },
    scheduling_tasks_hatchet: {
      start: 204,
      stop: 226,
    },
  },
  highlights: {},
};

export default snippet;
