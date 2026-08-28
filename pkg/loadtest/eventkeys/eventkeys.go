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
	EventKeyDag     EventKey = "load-test:dag-event"
)

const workflowNamePrefix = "load-test-"

const (
	WorkerName = workflowNamePrefix + "worker"

	WorkflowBatchName        = workflowNamePrefix + "batch"
	WorkflowDurableName      = workflowNamePrefix + "durable"
	WorkflowDurableChildName = workflowNamePrefix + "durable-child"
	WorkflowDagName          = workflowNamePrefix + "dag"
)

func WorkflowStandardName(i int) string {
	return fmt.Sprintf("%s%d", workflowNamePrefix, i)
}

var All = []EventKey{EventKeyDefault, EventKeyBatch, EventKeyDurable, EventKeyDag}

func IsKnown(key EventKey) bool {
	return slices.Contains(All, key)
}

func (k EventKey) Name() string {
	switch k {
	case EventKeyDefault:
		return "default"
	case EventKeyBatch:
		return "batch"
	case EventKeyDurable:
		return "durable"
	case EventKeyDag:
		return "dag"
	default:
		return string(k)
	}
}

func ByName(name string) (EventKey, bool) {
	for _, k := range All {
		if k.Name() == name {
			return k, true
		}
	}
	return "", false
}

func Names(keys []EventKey) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = k.Name()
	}
	return out
}

func (k EventKey) String() string {
	return string(k)
}
