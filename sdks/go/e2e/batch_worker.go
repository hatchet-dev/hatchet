//go:build e2e

package e2e

import (
	"strings"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// Batch task input/output types for e2e tests.

type SimpleInput struct {
	Message string `json:"Message"`
}

type SimpleOutput struct {
	TransformedMessage string `json:"TransformedMessage"`
}

type KeyedInput struct {
	Message string `json:"Message"`
	Group   string `json:"group"`
}

type KeyedOutput struct {
	BatchKey   string `json:"batchKey"`
	BatchSize  int    `json:"batchSize"`
	UniqueKeys int    `json:"uniqueKeys"`
	Uppercase  string `json:"uppercase"`
}

type KeyedIntervalOutput struct {
	BatchKey   string `json:"batchKey"`
	BatchSize  int    `json:"batchSize"`
	UniqueKeys int    `json:"uniqueKeys"`
	Payload    string `json:"payload"`
}

type LargeInput struct {
	Data string `json:"data"`
}

type LargeOutput struct {
	Received   bool `json:"received"`
	BatchSize  int  `json:"batchSize"`
	DataLength int  `json:"dataLength"`
}

type SingleInput struct {
	Message string `json:"Message"`
}

type SingleOutput struct {
	Original  string `json:"original"`
	BatchSize int    `json:"batchSize"`
}

type OrderedInput struct {
	Index int `json:"index"`
}

type OrderedOutput struct {
	Index int `json:"index"`
}

// Package-level vars holding the batch task instances (set by registerBatchWorkflows).
var (
	testBatchSimple        *hatchet.StandaloneBatchTask
	testBatchKeyed         *hatchet.StandaloneBatchTask
	testBatchKeyedInterval *hatchet.StandaloneBatchTask
	testBatchLarge         *hatchet.StandaloneBatchTask
	testBatchSingle        *hatchet.StandaloneBatchTask
	testBatchOrdered       *hatchet.StandaloneBatchTask
)

// batchRunId is set once in TestMain so all batch task names are unique per run.
var batchRunId string

func registerBatchWorkflows(client *hatchet.Client) {
	testBatchSimple = client.NewStandaloneBatchTask(
		"batch-e2e-simple-"+batchRunId,
		func(items []SimpleInput) ([]SimpleOutput, error) {
			out := make([]SimpleOutput, len(items))
			for i, inp := range items {
				out[i] = SimpleOutput{TransformedMessage: strings.ToUpper(inp.Message)}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(3),
		hatchet.WithBatchMaxInterval(200*time.Millisecond),
		hatchet.WithBatchRetries(0),
	)

	testBatchKeyed = client.NewStandaloneBatchTask(
		"batch-e2e-keyed-"+batchRunId,
		func(items []KeyedInput) ([]KeyedOutput, error) {
			uniqueGroups := map[string]struct{}{}
			for _, inp := range items {
				uniqueGroups[inp.Group] = struct{}{}
			}
			out := make([]KeyedOutput, len(items))
			for i, inp := range items {
				out[i] = KeyedOutput{
					BatchKey:   inp.Group,
					BatchSize:  len(items),
					UniqueKeys: len(uniqueGroups),
					Uppercase:  strings.ToUpper(inp.Message),
				}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(2),
		hatchet.WithBatchMaxInterval(200*time.Millisecond),
		hatchet.WithBatchGroupKey("input.group"),
		hatchet.WithBatchRetries(0),
	)

	testBatchKeyedInterval = client.NewStandaloneBatchTask(
		"batch-e2e-keyed-interval-"+batchRunId,
		func(items []KeyedInput) ([]KeyedIntervalOutput, error) {
			uniqueGroups := map[string]struct{}{}
			for _, inp := range items {
				uniqueGroups[inp.Group] = struct{}{}
			}
			out := make([]KeyedIntervalOutput, len(items))
			for i, inp := range items {
				out[i] = KeyedIntervalOutput{
					BatchKey:   inp.Group,
					BatchSize:  len(items),
					UniqueKeys: len(uniqueGroups),
					Payload:    inp.Message,
				}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(3),
		hatchet.WithBatchMaxInterval(150*time.Millisecond),
		hatchet.WithBatchGroupKey("input.group"),
		hatchet.WithBatchRetries(0),
	)

	testBatchLarge = client.NewStandaloneBatchTask(
		"batch-e2e-large-"+batchRunId,
		func(items []LargeInput) ([]LargeOutput, error) {
			out := make([]LargeOutput, len(items))
			for i, inp := range items {
				out[i] = LargeOutput{
					Received:   true,
					BatchSize:  len(items),
					DataLength: len(inp.Data),
				}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(10),
		hatchet.WithBatchMaxInterval(1000*time.Second),
		hatchet.WithBatchRetries(0),
	)

	testBatchSingle = client.NewStandaloneBatchTask(
		"batch-e2e-single-"+batchRunId,
		func(items []SingleInput) ([]SingleOutput, error) {
			out := make([]SingleOutput, len(items))
			for i, inp := range items {
				out[i] = SingleOutput{Original: inp.Message, BatchSize: len(items)}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(1),
		hatchet.WithBatchMaxInterval(100*time.Millisecond),
		hatchet.WithBatchRetries(0),
	)

	testBatchOrdered = client.NewStandaloneBatchTask(
		"batch-e2e-ordered-"+batchRunId,
		func(items []OrderedInput) ([]OrderedOutput, error) {
			out := make([]OrderedOutput, len(items))
			for i, inp := range items {
				out[i] = OrderedOutput{Index: inp.Index}
			}
			return out, nil
		},
		hatchet.WithBatchMaxSize(20),
		hatchet.WithBatchMaxInterval(2000*time.Millisecond),
		hatchet.WithBatchRetries(0),
	)
}
