package contracts

import v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"

// Type aliases for backwards compatibility after these types moved to v1/shared/trigger.proto.
// Go type aliases are transparent — contracts.X IS v1.X, no conversion needed.
type TriggerWorkflowRequest = v1.TriggerWorkflowRequest
type DesiredWorkerLabels = v1.DesiredWorkerLabels
type WorkerLabelComparator = v1.WorkerLabelComparator
