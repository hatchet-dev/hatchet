package eventkeys

import (
	"fmt"
	"slices"
)

type EventKey string

const (
	EventKeyDefault EventKey = "load-test:event"
	EventKeyBatch   EventKey = "load-test:batch-event"
	EventKeyDurable EventKey = "load-test:durable-event"
)

const workflowNamePrefix = "load-test-"

const (
	WorkerName = workflowNamePrefix + "worker"

	WorkflowBatchName        = workflowNamePrefix + "batch"
	WorkflowDurableName      = workflowNamePrefix + "durable"
	WorkflowDurableChildName = workflowNamePrefix + "durable-child"
)

func WorkflowStandardName(i int) string {
	return fmt.Sprintf("%s%d", workflowNamePrefix, i)
}

var All = []EventKey{EventKeyDefault, EventKeyBatch, EventKeyDurable}

func IsKnown(key EventKey) bool {
	return slices.Contains(All, key)
}

func (k EventKey) String() string {
	return string(k)
}

func Strings(keys []EventKey) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = string(k)
	}
	return out
}
