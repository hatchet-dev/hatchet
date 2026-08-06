package v1alpha

import "github.com/hatchet-dev/hatchet/pkg/scheduling"

// These types cross the engine boundary (channels and optimistic-scheduling
// results), so they are defined once in pkg/scheduling and aliased here: the
// engine handles identical types no matter which scheduler implementation a
// shard runs.
type QueueResults = scheduling.QueueResults
type ConcurrencyResults = scheduling.ConcurrencyResults
type AssignedItemWithTask = scheduling.AssignedItemWithTask
