package limits

import "time"

type LimitConfigFile struct {
	DefaultTenantRetentionPeriod string `mapstructure:"defaultTenantRetentionPeriod" json:"defaultTenantRetentionPeriod,omitempty" default:"720h"`
	CorePartitionRetention       string `mapstructure:"corePartitionRetention" json:"corePartitionRetention,omitempty"`
	OLAPPartitionRetention       string `mapstructure:"olapPartitionRetention" json:"olapPartitionRetention,omitempty"`

	DefaultTaskRunLimit      int32         `mapstructure:"defaultTaskRunLimit" json:"defaultTaskRunLimit,omitempty" default:"2000"`
	DefaultTaskRunAlarmLimit int32         `mapstructure:"defaultTaskRunAlarmLimit" json:"defaultTaskRunAlarmLimit,omitempty" default:"1600"`
	DefaultTaskRunWindow     time.Duration `mapstructure:"defaultTaskRunWindow" json:"defaultTaskRunWindow,omitempty" default:"24h"`

	DefaultWorkerLimit      int32 `mapstructure:"defaultWorkerLimit" json:"defaultWorkerLimit,omitempty" default:"3"`
	DefaultWorkerAlarmLimit int32 `mapstructure:"defaultWorkerAlarmLimit" json:"defaultWorkerAlarmLimit,omitempty" default:"2"`

	DefaultWorkerSlotLimit      int32 `mapstructure:"defaultWorkerSlotLimit" json:"defaultWorkerSlotLimit,omitempty" default:"2000"`
	DefaultWorkerSlotAlarmLimit int32 `mapstructure:"defaultWorkerSlotAlarmLimit" json:"defaultWorkerSlotAlarmLimit,omitempty" default:"1600"`

	DefaultEventLimit      int32         `mapstructure:"defaultEventLimit" json:"defaultEventLimit,omitempty" default:"1000"`
	DefaultEventAlarmLimit int32         `mapstructure:"defaultEventAlarmLimit" json:"defaultEventAlarmLimit,omitempty" default:"800"`
	DefaultEventWindow     time.Duration `mapstructure:"defaultEventWindow" json:"defaultEventWindow,omitempty" default:"24h"`

	DefaultIncomingWebhookLimit      int32 `mapstructure:"defaultIncomingWebhookLimit" json:"defaultIncomingWebhookLimit,omitempty" default:"5"`
	DefaultIncomingWebhookAlarmLimit int32 `mapstructure:"defaultIncomingWebhookAlarmLimit" json:"defaultIncomingWebhookALarmLimit,omitempty" default:"4"`
}

// CorePartitionRetentionOrDefault returns the core partition retention override or the tenant default.
func (c LimitConfigFile) CorePartitionRetentionOrDefault() string {
	if c.CorePartitionRetention != "" {
		return c.CorePartitionRetention
	}

	return c.DefaultTenantRetentionPeriod
}

// OLAPPartitionRetentionOrDefault returns the OLAP partition retention override or the tenant default.
func (c LimitConfigFile) OLAPPartitionRetentionOrDefault() string {
	if c.OLAPPartitionRetention != "" {
		return c.OLAPPartitionRetention
	}

	return c.DefaultTenantRetentionPeriod
}
