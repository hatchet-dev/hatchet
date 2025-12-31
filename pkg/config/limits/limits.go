package limits

import "time"

type LimitConfigFile struct {
	DefaultTenantRetentionPeriod string `mapstructure:"defaultTenantRetentionPeriod" json:"defaultTenantRetentionPeriod,omitempty" default:"720h"`

	DefaultWorkflowRunLimit      int           `mapstructure:"defaultWorkflowRunLimit" json:"defaultWorkflowRunLimit,omitempty" default:"1000"`
	DefaultWorkflowRunAlarmLimit int           `mapstructure:"defaultWorkflowRunAlarmLimit" json:"defaultWorkflowRunAlarmLimit,omitempty" default:"750"`
	DefaultWorkflowRunWindow     time.Duration `mapstructure:"defaultWorkflowRunWindow" json:"defaultWorkflowRunWindow,omitempty" default:"24h"`

	DefaultTaskRunLimit      int           `mapstructure:"defaultTaskRunLimit" json:"defaultTaskRunLimit,omitempty" default:"2000"`
	DefaultTaskRunAlarmLimit int           `mapstructure:"defaultTaskRunAlarmLimit" json:"defaultTaskRunAlarmLimit,omitempty" default:"1500"`
	DefaultTaskRunWindow     time.Duration `mapstructure:"defaultTaskRunWindow" json:"defaultTaskRunWindow,omitempty" default:"24h"`

	DefaultWorkerLimit      int `mapstructure:"defaultWorkerLimit" json:"defaultWorkerLimit,omitempty" default:"4"`
	DefaultWorkerAlarmLimit int `mapstructure:"defaultWorkerAlarmLimit" json:"defaultWorkerAlarmLimit,omitempty" default:"2"`

	DefaultWorkerSlotLimit      int `mapstructure:"defaultWorkerSlotLimit" json:"defaultWorkerSlotLimit,omitempty" default:"4000"`
	DefaultWorkerSlotAlarmLimit int `mapstructure:"defaultWorkerSlotAlarmLimit" json:"defaultWorkerSlotAlarmLimit,omitempty" default:"3000"`

	DefaultEventLimit      int           `mapstructure:"defaultEventLimit" json:"defaultEventLimit,omitempty" default:"1000"`
	DefaultEventAlarmLimit int           `mapstructure:"defaultEventAlarmLimit" json:"defaultEventAlarmLimit,omitempty" default:"750"`
	DefaultEventWindow     time.Duration `mapstructure:"defaultEventWindow" json:"defaultEventWindow,omitempty" default:"24h"`

	DefaultCronLimit      int `mapstructure:"defaultCronLimit" json:"defaultCronLimit,omitempty" default:"5"`
	DefaultCronAlarmLimit int `mapstructure:"defaultCronAlarmLimit" json:"defaultCronAlarmLimit,omitempty" default:"2"`

	DefaultScheduleLimit      int `mapstructure:"defaultScheduleLimit" json:"defaultScheduleLimit,omitempty" default:"1000"`
	DefaultScheduleAlarmLimit int `mapstructure:"defaultScheduleAlarmLimit" json:"defaultScheduleAlarmLimit,omitempty" default:"750"`

	DefaultIncomingWebhookLimit      int `mapstructure:"defaultIncomingWebhookLimit" json:"defaultIncomingWebhookLimit,omitempty" default:"5"`
	DefaultIncomingWebhookAlarmLimit int `mapstructure:"defaultIncomingWebhookAlarmLimit" json:"defaultIncomingWebhookALarmLimit,omitempty" default:"4"`
}
