package hatchet

import "github.com/hatchet-dev/hatchet/pkg/client/types"

// BatchMemberId identifies a single item within a batch task's input/output map. Its value
// is the external id of the buffered item's underlying task run.
type BatchMemberId = string

// BatchConfig configures batching behavior for a batch task. See Workflow.NewBatchTask.
type BatchConfig = types.BatchConfig
