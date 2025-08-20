package analytics

type Analytics interface {
	// Enqueue queues an analytics event for processing
	// @param event The name of the event to track
	// @param userId The ID of the user performing the action
	// @param tenantId Optional tenant ID to associate with this event
	// @param set Key-value pairs to set on the user/group profile (e.g. email, name, etc.)
	// @param metadata Additional metadata to attach to the event
	Enqueue(event string, userId string, tenantId *string, set map[string]interface{}, metadata map[string]interface{})

	// Tenant updates properties for a tenant group
	// @param tenantId The ID of the tenant to update
	// @param data Key-value pairs of properties to set on the tenant
	Tenant(tenantId string, data map[string]interface{})
}

type NoOpAnalytics struct{}

func (a NoOpAnalytics) Enqueue(event string, userId string, tenantId *string, set map[string]interface{}, metadata map[string]interface{}) {
}

func (a NoOpAnalytics) Tenant(tenantId string, data map[string]interface{}) {}
