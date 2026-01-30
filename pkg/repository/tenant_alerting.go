package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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
	TenantId uuid.UUID `json:"tenantId"`
	Emails   []string  `validate:"required,dive,email,max=255"`
}

type GetTenantAlertingSettingsResponse struct {
	Settings *sqlcv1.TenantAlertingSettings

	SlackWebhooks []*sqlcv1.SlackAppWebhook

	EmailGroups []*TenantAlertEmailGroupForSend

	Tenant *sqlcv1.Tenant
}

type TenantAlertingRepository interface {
	UpsertTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpsertTenantAlertingSettingsOpts) (*sqlcv1.TenantAlertingSettings, error)

	GetTenantAlertingSettings(ctx context.Context, tenantId string) (*GetTenantAlertingSettingsResponse, error)

	GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*sqlcv1.GetTenantResourceLimitRow, error)

	UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpdateTenantAlertingSettingsOpts) error

	CreateTenantAlertGroup(ctx context.Context, tenantId string, opts *CreateTenantAlertGroupOpts) (*sqlcv1.TenantAlertEmailGroup, error)

	UpdateTenantAlertGroup(ctx context.Context, id string, opts *UpdateTenantAlertGroupOpts) (*sqlcv1.TenantAlertEmailGroup, error)

	ListTenantAlertGroups(ctx context.Context, tenantId string) ([]*sqlcv1.TenantAlertEmailGroup, error)

	GetTenantAlertGroupById(ctx context.Context, id string) (*sqlcv1.TenantAlertEmailGroup, error)

	DeleteTenantAlertGroup(ctx context.Context, tenantId string, id string) error
}

type tenantAlertingRepository struct {
	*sharedRepository

	cache cache.Cacheable
}

func newTenantAlertingRepository(shared *sharedRepository, cacheDuration time.Duration) TenantAlertingRepository {
	return &tenantAlertingRepository{
		sharedRepository: shared,
		cache:            cache.New(cacheDuration),
	}
}

func (r *tenantAlertingRepository) UpsertTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpsertTenantAlertingSettingsOpts) (*sqlcv1.TenantAlertingSettings, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpsertTenantAlertingSettingsParams{
		Tenantid: uuid.MustParse(tenantId),
	}

	if opts.MaxFrequency != nil {
		params.MaxFrequency = sqlchelpers.TextFromStr(*opts.MaxFrequency)
	}

	if opts.EnableExpiringTokenAlerts != nil {
		params.EnableExpiringTokenAlerts = sqlchelpers.BoolFromBoolean(*opts.EnableExpiringTokenAlerts)
	}

	if opts.EnableWorkflowRunFailureAlerts != nil {
		params.EnableWorkflowRunFailureAlerts = sqlchelpers.BoolFromBoolean(*opts.EnableWorkflowRunFailureAlerts)
	}

	if opts.EnableTenantResourceLimitAlerts != nil {
		params.EnableTenantResourceLimitAlerts = sqlchelpers.BoolFromBoolean(*opts.EnableTenantResourceLimitAlerts)
	}

	return r.queries.UpsertTenantAlertingSettings(
		ctx,
		r.pool,
		params,
	)
}

func (r *tenantAlertingRepository) CreateTenantAlertGroup(ctx context.Context, tenantId string, opts *CreateTenantAlertGroupOpts) (*sqlcv1.TenantAlertEmailGroup, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.queries.CreateTenantAlertGroup(
		ctx,
		r.pool,
		sqlcv1.CreateTenantAlertGroupParams{
			Tenantid: uuid.MustParse(tenantId),
			Emails:   emails,
		},
	)
}

func (r *tenantAlertingRepository) UpdateTenantAlertGroup(ctx context.Context, id string, opts *UpdateTenantAlertGroupOpts) (*sqlcv1.TenantAlertEmailGroup, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.queries.UpdateTenantAlertGroup(
		ctx,
		r.pool,
		sqlcv1.UpdateTenantAlertGroupParams{
			ID:     uuid.MustParse(id),
			Emails: emails,
		},
	)
}

func (r *tenantAlertingRepository) ListTenantAlertGroups(ctx context.Context, tenantId string) ([]*sqlcv1.TenantAlertEmailGroup, error) {
	return r.queries.ListTenantAlertGroups(
		ctx,
		r.pool,
		uuid.MustParse(tenantId),
	)
}

func (r *tenantAlertingRepository) GetTenantAlertGroupById(ctx context.Context, id string) (*sqlcv1.TenantAlertEmailGroup, error) {
	return r.queries.GetTenantAlertGroupById(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}

func (r *tenantAlertingRepository) DeleteTenantAlertGroup(ctx context.Context, tenantId string, id string) error {
	return r.queries.DeleteTenantAlertGroup(
		ctx,
		r.pool,
		sqlcv1.DeleteTenantAlertGroupParams{
			Tenantid: uuid.MustParse(tenantId),
			ID:       uuid.MustParse(id),
		},
	)
}

func (r *tenantAlertingRepository) GetTenantAlertingSettings(ctx context.Context, tenantId string) (*GetTenantAlertingSettingsResponse, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	pgTenantId := uuid.MustParse(tenantId)

	settings, err := r.queries.GetTenantAlertingSettings(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	webhooks, err := r.queries.GetSlackWebhooks(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	groupsForSend := make([]*TenantAlertEmailGroupForSend, 0)

	emailGroups, err := r.queries.GetEmailGroups(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	for _, group := range emailGroups {
		emails := strings.Split(group.Emails, ",")

		groupsForSend = append(groupsForSend, &TenantAlertEmailGroupForSend{
			TenantId: group.TenantId,
			Emails:   emails,
		})
	}

	tenant, err := r.queries.GetTenantByID(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	if tenant.AlertMemberEmails {
		emails, err := r.queries.GetMemberEmailGroup(ctx, tx, pgTenantId)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				r.l.Warn().Err(err).Msg("No valid member email group found for tenant")
			} else {
				return nil, err
			}
		} else {
			groupsForSend = append(groupsForSend, &TenantAlertEmailGroupForSend{
				TenantId: tenant.ID,
				Emails:   emails,
			})
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return &GetTenantAlertingSettingsResponse{
		Settings:      settings,
		SlackWebhooks: webhooks,
		EmailGroups:   groupsForSend,
		Tenant:        tenant,
	}, nil
}

func (r *tenantAlertingRepository) UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *UpdateTenantAlertingSettingsOpts) error {
	if err := r.v.Validate(opts); err != nil {
		return err
	}

	updateParams := sqlcv1.UpdateTenantAlertingSettingsParams{
		TenantId: uuid.MustParse(tenantId),
	}

	if opts.LastAlertedAt != nil {
		updateParams.LastAlertedAt = sqlchelpers.TimestampFromTime(*opts.LastAlertedAt)
	}

	_, err := r.queries.UpdateTenantAlertingSettings(
		ctx,
		r.pool,
		updateParams,
	)

	return err
}

func (r *tenantAlertingRepository) GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*sqlcv1.GetTenantResourceLimitRow, error) {
	return r.queries.GetTenantResourceLimit(ctx, r.pool, sqlcv1.GetTenantResourceLimitParams{
		Tenantid: uuid.MustParse(tenantId),
		Resource: sqlcv1.NullLimitResource{
			LimitResource: sqlcv1.LimitResource(resource),
			Valid:         true,
		},
	})
}
