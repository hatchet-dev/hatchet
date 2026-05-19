package limits

import "time"

type LimitConfigFile struct {
	DefaultTenantRetentionPeriod string `mapstructure:"defaultTenantRetentionPeriod" json:"defaultTenantRetentionPeriod,omitempty" default:"720h"`

	DefaultWorkflowRunLimit      int32         `mapstructure:"defaultWorkflowRunLimit" json:"defaultWorkflowRunLimit,omitempty" default:"2000"`
	DefaultWorkflowRunAlarmLimit int32         `mapstructure:"defaultWorkflowRunAlarmLimit" json:"defaultWorkflowRunAlarmLimit,omitempty" default:"1600"`
	DefaultWorkflowRunWindow     time.Duration `mapstructure:"defaultWorkflowRunWindow" json:"defaultWorkflowRunWindow,omitempty" default:"24h"`

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

	DefaultCronLimit      int32 `mapstructure:"defaultCronLimit" json:"defaultCronLimit,omitempty" default:"5"`
	DefaultCronAlarmLimit int32 `mapstructure:"defaultCronAlarmLimit" json:"defaultCronAlarmLimit,omitempty" default:"2"`

	DefaultScheduleLimit      int32 `mapstructure:"defaultScheduleLimit" json:"defaultScheduleLimit,omitempty" default:"1000"`
	DefaultScheduleAlarmLimit int32 `mapstructure:"defaultScheduleAlarmLimit" json:"defaultScheduleAlarmLimit,omitempty" default:"750"`

	DefaultIncomingWebhookLimit      int32 `mapstructure:"defaultIncomingWebhookLimit" json:"defaultIncomingWebhookLimit,omitempty" default:"5"`
	DefaultIncomingWebhookAlarmLimit int32 `mapstructure:"defaultIncomingWebhookAlarmLimit" json:"defaultIncomingWebhookALarmLimit,omitempty" default:"4"`
}
