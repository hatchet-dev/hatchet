package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"

	"github.com/jackc/pgx/v5/pgtype"
)

type UpsertTenantAlertingSettingsOpts struct {
	MaxFrequency                    *string `validate:"omitnil,duration"`
	EnableExpiringTokenAlerts       *bool   `validate:"omitnil"`
	EnableWorkflowRunFailureAlerts  *bool   `validate:"omitnil"`
	EnableTenantResourceLimitAlerts *bool   `validate:"omitnil"`
}

type UpdateTenantAlertingSettingsOpts struct {
	LastAlertedAt *time.Time
}

type CreateTenantAlertGroupOpts struct {
	Emails []string `validate:"required,dive,email,max=255"`
}

type UpdateTenantAlertGroupOpts struct {
	Emails []string `validate:"required,dive,email,max=255"`
}

type TenantAlertEmailGroupForSend struct {
	TenantId pgtype.UUID `json:"tenantId"`
	Emails   []string    `validate:"required,dive,email,max=255"`
}

type GetTenantAlertingSettingsResponse struct {
	Settings *dbsqlc.TenantAlertingSettings

	SlackWebhooks []*dbsqlc.SlackAppWebhook

	EmailGroups []*TenantAlertEmailGroupForSend

	Tenant *dbsqlc.Tenant
}

type TenantAlertingRepository interface {
	UpsertTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpsertTenantAlertingSettingsOpts) (*dbsqlc.TenantAlertingSettings, error)

	GetTenantAlertingSettings(ctx context.Context, tenantId string) (*GetTenantAlertingSettingsResponse, error)

	GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*dbsqlc.GetTenantResourceLimitRow, error)

	UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpdateTenantAlertingSettingsOpts) error

	CreateTenantAlertGroup(ctx context.Context, tenantId string, opts *CreateTenantAlertGroupOpts) (*dbsqlc.TenantAlertEmailGroup, error)

	UpdateTenantAlertGroup(ctx context.Context, id string, opts *UpdateTenantAlertGroupOpts) (*dbsqlc.TenantAlertEmailGroup, error)

	ListTenantAlertGroups(ctx context.Context, tenantId string) ([]*dbsqlc.TenantAlertEmailGroup, error)

	GetTenantAlertGroupById(ctx context.Context, id string) (*dbsqlc.TenantAlertEmailGroup, error)

	DeleteTenantAlertGroup(ctx context.Context, tenantId string, id string) error
}
