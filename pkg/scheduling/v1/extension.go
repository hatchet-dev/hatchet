package v1

import "github.com/hatchet-dev/hatchet/pkg/scheduling"

// The extension contract crosses the engine boundary — cloud and OSS register
// extensions against the pool — so it is defined once in pkg/scheduling and
// aliased here.
type PostAssignInput = scheduling.PostAssignInput
type SnapshotInput = scheduling.SnapshotInput
type SlotUtilization = scheduling.SlotUtilization
type WorkerCp = scheduling.WorkerCp
type SlotCp = scheduling.SlotCp
type SchedulerExtension = scheduling.SchedulerExtension
type Extensions = scheduling.Extensions
