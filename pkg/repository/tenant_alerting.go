package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

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

type TenantAlertingAPIRepository interface {
	UpsertTenantAlertingSettings(tenantId string, opts *UpsertTenantAlertingSettingsOpts) (*db.TenantAlertingSettingsModel, error)

	GetTenantAlertingSettings(tenantId string) (*db.TenantAlertingSettingsModel, error)

	CreateTenantAlertGroup(tenantId string, opts *CreateTenantAlertGroupOpts) (*db.TenantAlertEmailGroupModel, error)

	UpdateTenantAlertGroup(id string, opts *UpdateTenantAlertGroupOpts) (*db.TenantAlertEmailGroupModel, error)

	ListTenantAlertGroups(tenantId string) ([]db.TenantAlertEmailGroupModel, error)

	GetTenantAlertGroupById(id string) (*db.TenantAlertEmailGroupModel, error)

	DeleteTenantAlertGroup(tenantId string, id string) error
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

type TenantAlertingEngineRepository interface {
	GetTenantAlertingSettings(ctx context.Context, tenantId string) (*GetTenantAlertingSettingsResponse, error)

	UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpdateTenantAlertingSettingsOpts) error

	GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*dbsqlc.GetTenantResourceLimitRow, error)
}
