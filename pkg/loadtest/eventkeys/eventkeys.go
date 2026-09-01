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
	// EventKeyDagShapes triggers the DagShapeWorkflowNames set - a handful of
	// multi-step DAGs whose topologies (dense fan-in, conditional skip fan-out,
	// on-failure handler, deep retry chain) are modelled on real production
	// workloads, for exercising the DAG operator.
	EventKeyDagShapes EventKey = "load-test:dag-shapes-event"
	// EventKeyDagNested triggers the nested-DAG scenario: a parent DAG whose
	// fan-out task spawns N child DAG runs and waits for them (see
	// DagNestedWorkflowNames).
	EventKeyDagNested EventKey = "load-test:dag-nested-event"
)

const workflowNamePrefix = "load-test-"

const (
	WorkerName = workflowNamePrefix + "worker"

	WorkflowBatchName        = workflowNamePrefix + "batch"
	WorkflowDurableName      = workflowNamePrefix + "durable"
	WorkflowDurableChildName = workflowNamePrefix + "durable-child"
	WorkflowDagName          = workflowNamePrefix + "dag"

	// The EventKeyDagShapes workflows. Each is a standalone DAG with a distinct
	// topology; all four are triggered by a single EventKeyDagShapes event.
	WorkflowDagShapeDenseFaninName  = workflowNamePrefix + "dag-shape-dense-fanin"
	WorkflowDagShapeConditionalName = workflowNamePrefix + "dag-shape-conditional"
	WorkflowDagShapeOnFailureName   = workflowNamePrefix + "dag-shape-onfailure"
	WorkflowDagShapeDeepRetryName   = workflowNamePrefix + "dag-shape-deep-retry"

	// The EventKeyDagNested workflows. The parent is event-triggered; its
	// fan-out task spawns runs of the child, which is not event-triggered.
	WorkflowDagShapeNestedParentName = workflowNamePrefix + "dag-shape-nested-parent"
	WorkflowDagShapeNestedChildName  = workflowNamePrefix + "dag-shape-nested-child"
)

// DagShapeWorkflowNames is the ordered set of workflows triggered by
// EventKeyDagShapes.
var DagShapeWorkflowNames = []string{
	WorkflowDagShapeDenseFaninName,
	WorkflowDagShapeConditionalName,
	WorkflowDagShapeOnFailureName,
	WorkflowDagShapeDeepRetryName,
}

// DagNestedWorkflowNames is the parent + child pair for the nested-DAG scenario.
var DagNestedWorkflowNames = []string{
	WorkflowDagShapeNestedParentName,
	WorkflowDagShapeNestedChildName,
}

func WorkflowStandardName(i int) string {
	return fmt.Sprintf("%s%d", workflowNamePrefix, i)
}

var All = []EventKey{EventKeyDefault, EventKeyBatch, EventKeyDurable, EventKeyDag, EventKeyDagShapes, EventKeyDagNested}

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
	case EventKeyDagShapes:
		return "dag-shapes"
	case EventKeyDagNested:
		return "dag-nested"
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
