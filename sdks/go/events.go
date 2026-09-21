package hatchet

import (
	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck // SA1019: bridges the v0 event client into the new SDK surface
)

// EventClient sends events to Hatchet, triggering any workflows subscribed to the event key.
// Obtain one from Client.Events().
type EventClient = v0Client.EventClient

// PushOpFunc configures a single event pushed via EventClient.Push.
type PushOpFunc = v0Client.PushOpFunc

// BulkPushOpFunc configures a bulk event push via EventClient.BulkPush.
type BulkPushOpFunc = v0Client.BulkPushOpFunc

// EventWithAdditionalMetadata is a single event in an EventClient.BulkPush call.
type EventWithAdditionalMetadata = v0Client.EventWithAdditionalMetadata

// WithEventMetadata attaches additional metadata to the pushed event.
func WithEventMetadata(metadata map[string]string) PushOpFunc {
	return v0Client.WithEventMetadata(metadata)
}

// WithEventPriority sets the priority of the runs triggered by the pushed event.
func WithEventPriority(priority *int32) PushOpFunc {
	return v0Client.WithEventPriority(priority)
}

// WithFilterScope sets the filter scope for the pushed event, matching it against
// default filters declared with the same scope (see WithDefaultFilters).
func WithFilterScope(scope *string) PushOpFunc {
	return v0Client.WithFilterScope(scope)
}
