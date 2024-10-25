package prisma

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantAlertingAPIRepository struct {
	client *db.PrismaClient
	v      validator.Validator
	cache  cache.Cacheable
}

func NewTenantAlertingAPIRepository(client *db.PrismaClient, v validator.Validator, cache cache.Cacheable) repository.TenantAlertingAPIRepository {
	return &tenantAlertingAPIRepository{
		client: client,
		v:      v,
		cache:  cache,
	}
}

func (r *tenantAlertingAPIRepository) UpsertTenantAlertingSettings(tenantId string, opts *repository.UpsertTenantAlertingSettingsOpts) (*db.TenantAlertingSettingsModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.TenantAlertingSettings.UpsertOne(
		db.TenantAlertingSettings.TenantID.Equals(tenantId),
	).Create(
		db.TenantAlertingSettings.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.TenantAlertingSettings.MaxFrequency.SetIfPresent(opts.MaxFrequency),
		db.TenantAlertingSettings.EnableExpiringTokenAlerts.SetIfPresent(opts.EnableExpiringTokenAlerts),
		db.TenantAlertingSettings.EnableWorkflowRunFailureAlerts.SetIfPresent(opts.EnableWorkflowRunFailureAlerts),
		db.TenantAlertingSettings.EnableTenantResourceLimitAlerts.SetIfPresent(opts.EnableTenantResourceLimitAlerts),
	).Update(
		db.TenantAlertingSettings.MaxFrequency.SetIfPresent(opts.MaxFrequency),
		db.TenantAlertingSettings.EnableExpiringTokenAlerts.SetIfPresent(opts.EnableExpiringTokenAlerts),
		db.TenantAlertingSettings.EnableWorkflowRunFailureAlerts.SetIfPresent(opts.EnableWorkflowRunFailureAlerts),
		db.TenantAlertingSettings.EnableTenantResourceLimitAlerts.SetIfPresent(opts.EnableTenantResourceLimitAlerts),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) GetTenantAlertingSettings(tenantId string) (*db.TenantAlertingSettingsModel, error) {
	return r.client.TenantAlertingSettings.FindFirst(
		db.TenantAlertingSettings.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) CreateTenantAlertGroup(tenantId string, opts *repository.CreateTenantAlertGroupOpts) (*db.TenantAlertEmailGroupModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.client.TenantAlertEmailGroup.CreateOne(
		db.TenantAlertEmailGroup.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.TenantAlertEmailGroup.Emails.Set(emails),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) UpdateTenantAlertGroup(id string, opts *repository.UpdateTenantAlertGroupOpts) (*db.TenantAlertEmailGroupModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	emails := strings.Join(opts.Emails, ",")

	return r.client.TenantAlertEmailGroup.FindUnique(
		db.TenantAlertEmailGroup.ID.Equals(id),
	).Update(
		db.TenantAlertEmailGroup.Emails.Set(emails),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) ListTenantAlertGroups(tenantId string) ([]db.TenantAlertEmailGroupModel, error) {
	return r.client.TenantAlertEmailGroup.FindMany(
		db.TenantAlertEmailGroup.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) GetTenantAlertGroupById(id string) (*db.TenantAlertEmailGroupModel, error) {
	return r.client.TenantAlertEmailGroup.FindUnique(
		db.TenantAlertEmailGroup.ID.Equals(id),
	).Exec(context.Background())
}

func (r *tenantAlertingAPIRepository) DeleteTenantAlertGroup(tenantId string, id string) error {
	_, err := r.client.TenantAlertEmailGroup.FindUnique(
		db.TenantAlertEmailGroup.ID.Equals(id),
	).Delete().Exec(context.Background())

	return err
}

type tenantAlertingEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewTenantAlertingEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cache cache.Cacheable) repository.TenantAlertingEngineRepository {
	queries := dbsqlc.New()

	return &tenantAlertingEngineRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (r *tenantAlertingEngineRepository) GetTenantAlertingSettings(ctx context.Context, tenantId string) (*repository.GetTenantAlertingSettingsResponse, error) {
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

func (r *tenantAlertingEngineRepository) UpdateTenantAlertingSettings(ctx context.Context, tenantId string, opts *repository.UpdateTenantAlertingSettingsOpts) error {
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

func (r *tenantAlertingEngineRepository) GetTenantResourceLimitState(ctx context.Context, tenantId string, resource string) (*dbsqlc.GetTenantResourceLimitRow, error) {
	return r.queries.GetTenantResourceLimit(ctx, r.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: dbsqlc.LimitResource(resource),
			Valid:         true,
		},
	})
}
