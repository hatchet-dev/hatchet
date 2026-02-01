package constants

type MetadataKey string

const (
	CorrelationIdKey  MetadataKey = "correlationId"
	ResourceIdKey     MetadataKey = "resourceId"
	ResourceTypeKey   MetadataKey = "resourceType"
	GRPCMethodKey     MetadataKey = "grpc_method"
	EventIDKey        MetadataKey = "hatchet__event_id"
	EventKeyKey       MetadataKey = "hatchet__event_key"
	CronExpressionKey MetadataKey = "hatchet__cron_expression"
	CronNameKey       MetadataKey = "hatchet__cron_name"
	TriggeredByKey    MetadataKey = "hatchet__triggered_by"
	ScheduledAtKey    MetadataKey = "hatchet__scheduled_at"
)

func (k MetadataKey) String() string {
	return string(k)
}

type ResourceTypeValue string

const (
	ResourceTypeApiToken          ResourceTypeValue = "api-token"
	ResourceTypeTenantMember      ResourceTypeValue = "tenant-member"
	ResourceTypeTenantInvite      ResourceTypeValue = "tenant-invite"
	ResourceTypeWorkflow          ResourceTypeValue = "workflow"
	ResourceTypeWorkflowRun       ResourceTypeValue = "workflow-run"
	ResourceTypeScheduledWorkflow ResourceTypeValue = "scheduled-workflow"
	ResourceTypeEvent             ResourceTypeValue = "event"
)

func (k ResourceTypeValue) String() string {
	return string(k)
}
