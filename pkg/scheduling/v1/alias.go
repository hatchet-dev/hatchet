package v1

import "github.com/hatchet-dev/hatchet/pkg/scheduling"

// These types cross the engine boundary (channels and optimistic-scheduling
// results), so they are defined once in pkg/scheduling and aliased here.
type QueueResults = scheduling.QueueResults
type ConcurrencyResults = scheduling.ConcurrencyResults
type AssignedItemWithTask = scheduling.AssignedItemWithTask
