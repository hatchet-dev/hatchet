package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type UpsertTenantAlertingSettingsOpts struct {
	MaxFrequency *string `validate:"omitnil,duration"`
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

type GetTenantAlertingSettingsResponse struct {
	Settings *dbsqlc.TenantAlertingSettings

	SlackWebhooks []*dbsqlc.SlackAppWebhook

	EmailGroups []*dbsqlc.TenantAlertEmailGroup

	Tenant *dbsqlc.Tenant
}

type TenantAlertingEngineRepository interface {
	GetTenantAlertingSettings(ctx context.Context, tenantId string) (*GetTenantAlertingSettingsResponse, error)

	UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpdateTenantAlertingSettingsOpts) error
}
