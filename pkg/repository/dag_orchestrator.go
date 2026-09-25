package repository

import (
	"regexp"
	"strings"
)

// dagOrchestratorActionSuffix ends the action id of the orchestrator step the engine adds to
// a DAG workflow when the tenant runs the DAG operator.
const dagOrchestratorActionSuffix = "_orchestrator"

// dagOrchestratorActionRegex is the form DAGOrchestratorActionId produces: a lower-cased
// workflow name (the hatchetName alphabet) followed by the suffix. It admits no ":", so it is
// disjoint from the "service:verb" action ids workers register, and no ";", the separator
// the worker action hash frames ids with.
var dagOrchestratorActionRegex = regexp.MustCompile(`^[a-z0-9.\-_]+_orchestrator$`)

// DAGOrchestratorActionId is the action id of the orchestrator step of the named workflow.
// It has no ":", so ParseActionID rejects it and no SDK or GRPC worker can register it: only
// the engine's DAG operator holds orchestrator actions, and the operator service admits them
// on DAG operator sessions alone (see IsDAGOrchestratorActionId).
func DAGOrchestratorActionId(workflowName string) string {
	return strings.ToLower(workflowName + dagOrchestratorActionSuffix)
}

// IsDAGOrchestratorActionId reports whether id is an orchestrator action id as
// DAGOrchestratorActionId produces it.
func IsDAGOrchestratorActionId(id string) bool {
	return dagOrchestratorActionRegex.MatchString(id)
}
