package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type tenantAlertingRepository struct {
	*sharedRepository

	cache cache.Cacheable
}

func NewTenantAlertingRepository(shared *sharedRepository, cache cache.Cacheable) repository.TenantAlertingRepository {
	return &tenantAlertingRepository{
		sharedRepository: shared,
		cache:            cache,
	}
}

func (r *tenantAlertingRepository) UpsertTenantAlertingSettings(ctx context.Context, tenantId string, opts *repository.UpsertTenantAlertingSettingsOpts) (*dbsqlc.TenantAlertingSettings, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpsertTenantAlertingSettingsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
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

func (r *tenantAlertingRepository) CreateTenantAlertGroup(ctx context.Context, tenantId string, opts *repository.CreateTenantAlertGroupOpts) (*dbsqlc.TenantAlertEmailGroup, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.queries.CreateTenantAlertGroup(
		ctx,
		r.pool,
		dbsqlc.CreateTenantAlertGroupParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Emails:   emails,
		},
	)
}

func (r *tenantAlertingRepository) UpdateTenantAlertGroup(ctx context.Context, id string, opts *repository.UpdateTenantAlertGroupOpts) (*dbsqlc.TenantAlertEmailGroup, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.queries.UpdateTenantAlertGroup(
		ctx,
		r.pool,
		dbsqlc.UpdateTenantAlertGroupParams{
			ID:     sqlchelpers.UUIDFromStr(id),
			Emails: emails,
		},
	)
}

func (r *tenantAlertingRepository) ListTenantAlertGroups(ctx context.Context, tenantId string) ([]*dbsqlc.TenantAlertEmailGroup, error) {
	return r.queries.ListTenantAlertGroups(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(tenantId),
	)
}

func (r *tenantAlertingRepository) GetTenantAlertGroupById(ctx context.Context, id string) (*dbsqlc.TenantAlertEmailGroup, error) {
	return r.queries.GetTenantAlertGroupById(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(id),
	)
}

func (r *tenantAlertingRepository) DeleteTenantAlertGroup(ctx context.Context, tenantId string, id string) error {
	return r.queries.DeleteTenantAlertGroup(
		ctx,
		r.pool,
		dbsqlc.DeleteTenantAlertGroupParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			ID:       sqlchelpers.UUIDFromStr(id),
		},
	)
}

func (r *tenantAlertingRepository) GetTenantAlertingSettings(ctx context.Context, tenantId string) (*repository.GetTenantAlertingSettingsResponse, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	settings, err := r.queries.GetTenantAlertingSettings(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	webhooks, err := r.queries.GetSlackWebhooks(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	groupsForSend := make([]*repository.TenantAlertEmailGroupForSend, 0)

	emailGroups, err := r.queries.GetEmailGroups(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	for _, group := range emailGroups {
		emails := strings.Split(group.Emails, ",")

		groupsForSend = append(groupsForSend, &repository.TenantAlertEmailGroupForSend{
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
			groupsForSend = append(groupsForSend, &repository.TenantAlertEmailGroupForSend{
				TenantId: tenant.ID,
				Emails:   emails,
			})
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return &repository.GetTenantAlertingSettingsResponse{
		Settings:      settings,
		SlackWebhooks: webhooks,
		EmailGroups:   groupsForSend,
		Tenant:        tenant,
	}, nil
}

func (r *tenantAlertingRepository) UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *repository.UpdateTenantAlertingSettingsOpts) error {
	if err := r.v.Validate(opts); err != nil {
		return err
	}

	updateParams := dbsqlc.UpdateTenantAlertingSettingsParams{
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
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

func (r *tenantAlertingRepository) GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*dbsqlc.GetTenantResourceLimitRow, error) {
	return r.queries.GetTenantResourceLimit(ctx, r.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: dbsqlc.LimitResource(resource),
			Valid:         true,
		},
	})
}
