//go:build e2e

package e2e

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// --- Batch task test types (mirrors sdks/python/examples/batch_assign/worker.py) ---

type OrderedInput struct {
	Index int `json:"index"`
}

type SimpleInput struct {
	Message string `json:"message"`
}

type KeyedInput struct {
	Message string `json:"message"`
	Group   string `json:"group"`
}

// KeyedFailableInput.Group is untyped so a caller can send a non-string group to trigger a
// server-side batch-group-key expression parse failure isolated to that run.
type KeyedFailableInput struct {
	Message string `json:"message"`
	Group   any    `json:"group"`
}

type LargePayloadInput struct {
	Data string `json:"data"`
}

type BroadcastOutput struct {
	Sum int `json:"sum"`
}

type ChildBatchOutput struct {
	Out map[string]SimpleInput `json:"out"`
}

type SimpleOutput struct {
	TransformedMessage string `json:"transformed_message"`
}

type KeyedOutput struct {
	BatchKey   *string `json:"batch_key,omitempty"`
	BatchSize  *int    `json:"batch_size,omitempty"`
	UniqueKeys *int    `json:"unique_keys,omitempty"`
	Uppercase  string  `json:"uppercase"`
}

type LargeOutput struct {
	BatchId    string `json:"batch_id"`
	Received   bool   `json:"received"`
	BatchSize  int    `json:"batch_size"`
	DataLength int    `json:"data_length"`
}

type SingleOutput struct {
	Original  string `json:"original"`
	BatchSize int    `json:"batch_size"`
}

type OrderedOutput struct {
	Index int `json:"index"`
}

type ChildOutput struct {
	MessageLen int `json:"message_len"`
}

// --- Batch task test workflow definitions ---

var (
	testBatchSimple          *hatchet.StandaloneTask
	testBatchKeyed           *hatchet.StandaloneTask
	testBatchKeyedFailable   *hatchet.StandaloneTask
	testBatchKeyedInterval   *hatchet.StandaloneTask
	testBatchLarge           *hatchet.StandaloneTask
	testBatchSingle          *hatchet.StandaloneTask
	testBatchOrdered         *hatchet.StandaloneTask
	testBatchBroadcast       *hatchet.StandaloneTask
	testBatchChild           *hatchet.StandaloneTask
	testBatchChildBatch      *hatchet.StandaloneTask
	testBatchChildSpawn      *hatchet.StandaloneTask
	testBatchChildBatchSpawn *hatchet.StandaloneTask
)

func registerBatchAssignWorkflows(client *hatchet.Client) {
	testBatchSimple = client.NewStandaloneBatchTask("batch-simple",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (map[string]SimpleOutput, error) {
			out := make(map[string]SimpleOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = SimpleOutput{TransformedMessage: strings.ToUpper(inp.Message)}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     3,
			MaxInterval: durationPtr(200 * time.Millisecond),
		},
	)

	testBatchKeyed = client.NewStandaloneBatchTask("batch-keyed",
		func(ctx hatchet.Context, tasks map[string]KeyedInput) (map[string]KeyedOutput, error) {
			return keyedOutputs(tasks), nil
		},
		hatchet.BatchConfig{
			MaxSize:     2,
			MaxInterval: durationPtr(200 * time.Millisecond),
			GroupKey:    stringPtr("input.group"),
		},
	)

	testBatchKeyedFailable = client.NewStandaloneBatchTask("batch-keyed-failable",
		func(ctx hatchet.Context, tasks map[string]KeyedFailableInput) (map[string]KeyedOutput, error) {
			out := make(map[string]KeyedOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = KeyedOutput{Uppercase: strings.ToUpper(inp.Message)}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     2,
			MaxInterval: durationPtr(200 * time.Millisecond),
			GroupKey:    stringPtr("input.group"),
		},
	)

	testBatchKeyedInterval = client.NewStandaloneBatchTask("batch-keyed-interval",
		func(ctx hatchet.Context, tasks map[string]KeyedInput) (map[string]KeyedOutput, error) {
			return keyedOutputs(tasks), nil
		},
		hatchet.BatchConfig{
			MaxSize:     3,
			MaxInterval: durationPtr(150 * time.Millisecond),
			GroupKey:    stringPtr("input.group"),
		},
	)

	testBatchLarge = client.NewStandaloneBatchTask("batch-large",
		func(ctx hatchet.Context, tasks map[string]LargePayloadInput) (map[string]LargeOutput, error) {
			batchID := uuid.New().String()
			out := make(map[string]LargeOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = LargeOutput{
					BatchId:    batchID,
					Received:   true,
					BatchSize:  len(tasks),
					DataLength: len(inp.Data),
				}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     100,
			MaxInterval: durationPtr(10 * time.Second),
		},
	)

	testBatchSingle = client.NewStandaloneBatchTask("batch-single",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (map[string]SingleOutput, error) {
			out := make(map[string]SingleOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = SingleOutput{Original: inp.Message, BatchSize: len(tasks)}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     1,
			MaxInterval: durationPtr(100 * time.Millisecond),
		},
	)

	testBatchOrdered = client.NewStandaloneBatchTask("batch-ordered",
		func(ctx hatchet.Context, tasks map[string]OrderedInput) (map[string]OrderedOutput, error) {
			out := make(map[string]OrderedOutput, len(tasks))
			for id, inp := range tasks {
				out[id] = OrderedOutput{Index: inp.Index}
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     20,
			MaxInterval: durationPtr(2 * time.Second),
		},
	)

	testBatchBroadcast = client.NewStandaloneBatchTask("batch-broadcast",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (BroadcastOutput, error) {
			sum := 0
			for _, inp := range tasks {
				sum += len(inp.Message)
			}
			return BroadcastOutput{Sum: sum}, nil
		},
		hatchet.BatchConfig{
			MaxSize:         10,
			MaxInterval:     durationPtr(2 * time.Second),
			BroadcastOutput: true,
		},
	)

	testBatchChild = client.NewStandaloneTask("batch-child",
		func(ctx hatchet.Context, input SimpleInput) (ChildOutput, error) {
			return ChildOutput{MessageLen: len(input.Message)}, nil
		},
	)

	testBatchChildBatch = client.NewStandaloneBatchTask("batch-child-batch",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (ChildBatchOutput, error) {
			return ChildBatchOutput{Out: tasks}, nil
		},
		hatchet.BatchConfig{
			MaxSize:         10,
			MaxInterval:     durationPtr(60 * time.Second),
			BroadcastOutput: true,
		},
	)

	testBatchChildSpawn = client.NewStandaloneBatchTask("batch-child-spawn",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (map[string]ChildOutput, error) {
			out := make(map[string]ChildOutput, len(tasks))
			for id := range tasks {
				result, err := testBatchChild.Run(ctx, SimpleInput{Message: "blahblah"})
				if err != nil {
					return nil, err
				}
				var childOut ChildOutput
				if err := result.Into(&childOut); err != nil {
					return nil, err
				}
				out[id] = childOut
			}
			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     10,
			MaxInterval: durationPtr(60 * time.Second),
		},
		hatchet.WithExecutionTimeout(60*time.Second),
	)

	testBatchChildBatchSpawn = client.NewStandaloneBatchTask("batch-child-batch-spawn",
		func(ctx hatchet.Context, tasks map[string]SimpleInput) (map[string]ChildBatchOutput, error) {
			out := make(map[string]ChildBatchOutput, len(tasks))
			var mu sync.Mutex
			var wg sync.WaitGroup
			var firstErr error
			var errMu sync.Mutex

			for id := range tasks {
				wg.Add(1)
				go func(id string) {
					defer wg.Done()

					result, err := testBatchChildBatch.Run(ctx, SimpleInput{Message: "hello"})
					if err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						return
					}

					var childOut ChildBatchOutput
					if err := result.Into(&childOut); err != nil {
						errMu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						errMu.Unlock()
						return
					}

					mu.Lock()
					out[id] = childOut
					mu.Unlock()
				}(id)
			}

			wg.Wait()

			if firstErr != nil {
				return nil, firstErr
			}

			return out, nil
		},
		hatchet.BatchConfig{
			MaxSize:     10,
			MaxInterval: durationPtr(60 * time.Second),
		},
		hatchet.WithExecutionTimeout(60*time.Second),
	)
}

func keyedOutputs(tasks map[string]KeyedInput) map[string]KeyedOutput {
	uniqueGroups := make(map[string]struct{})
	for _, inp := range tasks {
		uniqueGroups[inp.Group] = struct{}{}
	}
	uniqueKeys := len(uniqueGroups)
	batchSize := len(tasks)

	out := make(map[string]KeyedOutput, len(tasks))
	for id, inp := range tasks {
		group := inp.Group
		out[id] = KeyedOutput{
			BatchKey:   &group,
			BatchSize:  &batchSize,
			UniqueKeys: &uniqueKeys,
			Uppercase:  strings.ToUpper(inp.Message),
		}
	}
	return out
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func stringPtr(s string) *string {
	return &s
}
