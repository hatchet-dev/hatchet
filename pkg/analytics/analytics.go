package analytics

import "github.com/google/uuid"

type Analytics interface {
	// Enqueue queues an analytics event for processing.
	// event is the name of the event to track.
	// userId is the ID of the user performing the action.
	// tenantId is an optional tenant ID to associate with this event.
	// set contains key-value pairs to set on the user/group profile (e.g. email, name, etc.).
	// metadata contains additional metadata to attach to the event.
	Enqueue(event string, userId string, tenantId *uuid.UUID, set map[string]interface{}, metadata map[string]interface{})

	// Tenant updates properties for a tenant group.
	// tenantId is the ID of the tenant to update.
	// data contains key-value pairs of properties to set on the tenant.
	Tenant(tenantId uuid.UUID, data map[string]interface{})
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(event string, userId string, tenantId *uuid.UUID, set map[string]interface{}, metadata map[string]interface{}) {
}

func (a NoOpAnalytics) Tenant(tenantId uuid.UUID, data map[string]interface{}) {}
