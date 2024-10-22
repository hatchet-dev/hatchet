package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantAPIRepository struct {
	cache   cache.Cacheable
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewTenantAPIRepository(pool *pgxpool.Pool, client *db.PrismaClient, v validator.Validator, l *zerolog.Logger, cache cache.Cacheable) repository.TenantAPIRepository {
	queries := dbsqlc.New()

	return &tenantAPIRepository{
		cache:   cache,
		client:  client,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (r *tenantAPIRepository) CreateTenant(opts *repository.CreateTenantOpts) (*dbsqlc.Tenant, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	tenantId := uuid.New().String()

	if opts.ID != nil {
		tenantId = *opts.ID
	}

	var dataRetentionPeriod pgtype.Text

	if opts.DataRetentionPeriod != nil {
		dataRetentionPeriod = sqlchelpers.TextFromStr(*opts.DataRetentionPeriod)
	}

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	createTenant, err := r.queries.CreateTenant(context.Background(), tx, dbsqlc.CreateTenantParams{
		ID:                  sqlchelpers.UUIDFromStr(tenantId),
		Slug:                opts.Slug,
		Name:                opts.Name,
		DataRetentionPeriod: dataRetentionPeriod,
	})

	if err != nil {
		return nil, err
	}

	_, err = r.queries.CreateTenantAlertingSettings(context.Background(), tx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}

	return createTenant, nil
}

func (r *tenantAPIRepository) UpdateTenant(id string, opts *repository.UpdateTenantOpts) (*db.TenantModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.Tenant.FindUnique(
		db.Tenant.ID.Equals(id),
	).Update(
		db.Tenant.Name.SetIfPresent(opts.Name),
		db.Tenant.AnalyticsOptOut.SetIfPresent(opts.AnalyticsOptOut),
		db.Tenant.AlertMemberEmails.SetIfPresent(opts.AlertMemberEmails),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantByID(id string) (*db.TenantModel, error) {
	return cache.MakeCacheable[db.TenantModel](r.cache, "prisma"+id, func() (*db.TenantModel, error) {
		return r.client.Tenant.FindUnique(
			db.Tenant.ID.Equals(id),
		).Exec(context.Background())
	})
}

func (r *tenantAPIRepository) GetTenantBySlug(slug string) (*db.TenantModel, error) {
	return r.client.Tenant.FindUnique(
		db.Tenant.Slug.Equals(slug),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) CreateTenantMember(tenantId string, opts *repository.CreateTenantMemberOpts) (*db.TenantMemberModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.TenantMember.CreateOne(
		db.TenantMember.Tenant.Link(db.Tenant.ID.Equals(tenantId)),
		db.TenantMember.User.Link(db.User.ID.Equals(opts.UserId)),
		db.TenantMember.Role.Set(db.TenantMemberRole(opts.Role)),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByID(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByUserID(tenantId string, userId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.TenantIDUserID(
			db.TenantMember.TenantID.Equals(tenantId),
			db.TenantMember.UserID.Equals(userId),
		),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) ListTenantMembers(tenantId string) ([]db.TenantMemberModel, error) {
	return r.client.TenantMember.FindMany(
		db.TenantMember.TenantID.Equals(tenantId),
	).With(
		db.TenantMember.User.Fetch(),
		db.TenantMember.Tenant.Fetch(),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) GetTenantMemberByEmail(tenantId string, email string) (*db.TenantMemberModel, error) {
	user, err := r.client.User.FindUnique(
		db.User.Email.Equals(email),
	).Exec(context.Background())

	if err != nil {
		return nil, err
	}

	return r.client.TenantMember.FindUnique(
		db.TenantMember.TenantIDUserID(
			db.TenantMember.TenantID.Equals(tenantId),
			db.TenantMember.UserID.Equals(user.ID),
		),
	).Exec(context.Background())
}

func (r *tenantAPIRepository) UpdateTenantMember(memberId string, opts *repository.UpdateTenantMemberOpts) (*db.TenantMemberModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.TenantMemberSetParam{}

	if opts.Role != nil {
		params = append(params, db.TenantMember.Role.Set(db.TenantMemberRole(*opts.Role)))
	}

	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Update(
		params...,
	).Exec(context.Background())
}

func (r *tenantAPIRepository) DeleteTenantMember(memberId string) (*db.TenantMemberModel, error) {
	return r.client.TenantMember.FindUnique(
		db.TenantMember.ID.Equals(memberId),
	).Delete().Exec(context.Background())
}

func (r *tenantAPIRepository) GetQueueMetrics(ctx context.Context, tenantId string, opts *repository.GetQueueMetricsOpts) (*repository.GetQueueMetricsResponse, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	totalParams := dbsqlc.GetTenantTotalQueueMetricsParams{
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	}

	workflowParams := dbsqlc.GetTenantWorkflowQueueMetricsParams{
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.AdditionalMetadata != nil {
		additionalMetadataBytes, err := json.Marshal(opts.AdditionalMetadata)
		if err != nil {
			return nil, err
		}

		totalParams.AdditionalMetadata = additionalMetadataBytes
		workflowParams.AdditionalMetadata = additionalMetadataBytes
	}

	if opts.WorkflowIds != nil {
		uuids := make([]pgtype.UUID, len(opts.WorkflowIds))

		for i, id := range opts.WorkflowIds {
			uuids[i] = sqlchelpers.UUIDFromStr(id)
		}

		workflowParams.WorkflowIds = uuids
		totalParams.WorkflowIds = uuids
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 60*1000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// get the totals
	total, err := r.queries.GetTenantTotalQueueMetrics(ctx, tx, totalParams)

	if err != nil {
		return nil, err
	}

	// get the workflow metrics
	workflowMetrics, err := r.queries.GetTenantWorkflowQueueMetrics(ctx, tx, workflowParams)

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	workflowMetricsMap := make(map[string]repository.QueueMetric)

	for _, metric := range workflowMetrics {
		workflowMetricsMap[sqlchelpers.UUIDToStr(metric.WorkflowId)] = repository.QueueMetric{
			PendingAssignment: int(metric.PendingAssignmentCount),
			Pending:           int(metric.PendingCount),
			Running:           int(metric.RunningCount),
		}
	}

	return &repository.GetQueueMetricsResponse{
		Total: repository.QueueMetric{
			PendingAssignment: int(total.PendingAssignmentCount),
			Pending:           int(total.PendingCount),
			Running:           int(total.RunningCount),
		},
		ByWorkflowId: workflowMetricsMap,
	}, nil
}

type tenantEngineRepository struct {
	cache   cache.Cacheable
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewTenantEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cache cache.Cacheable) repository.TenantEngineRepository {
	queries := dbsqlc.New()

	return &tenantEngineRepository{
		cache:   cache,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (r *tenantEngineRepository) ListTenants(ctx context.Context) ([]*dbsqlc.Tenant, error) {
	return r.queries.ListTenants(ctx, r.pool)
}

func (r *tenantEngineRepository) GetTenantByID(ctx context.Context, tenantId string) (*dbsqlc.Tenant, error) {
	return cache.MakeCacheable[dbsqlc.Tenant](r.cache, tenantId, func() (*dbsqlc.Tenant, error) {
		return r.queries.GetTenantByID(ctx, r.pool, sqlchelpers.UUIDFromStr(tenantId))
	})
}

func (r *tenantEngineRepository) UpdateControllerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return "", err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, "SET statement_timeout=5000")

	if err != nil {
		return "", err
	}

	partition, err := r.queries.ControllerPartitionHeartbeat(ctx, tx, partitionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// create a new partition
			partition, err = r.queries.CreateControllerPartition(ctx, tx, getPartitionName())

			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) UpdateWorkerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return "", err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, "SET statement_timeout=5000")

	if err != nil {
		return "", err
	}

	partition, err := r.queries.WorkerPartitionHeartbeat(ctx, tx, partitionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// create a new partition
			partition, err = r.queries.CreateTenantWorkerPartition(ctx, tx, getPartitionName())

			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string) ([]*dbsqlc.Tenant, error) {
	if controllerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByControllerPartitionId(ctx, r.pool, controllerPartitionId)
}

func (r *tenantEngineRepository) ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string) ([]*dbsqlc.Tenant, error) {
	if workerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByTenantWorkerPartitionId(ctx, r.pool, workerPartitionId)
}

func (r *tenantEngineRepository) CreateControllerPartition(ctx context.Context) (string, error) {

	partition, err := r.queries.CreateControllerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) DeleteControllerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteControllerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantEngineRepository) RebalanceAllControllerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllControllerPartitions(ctx, r.pool)
}

func (r *tenantEngineRepository) RebalanceInactiveControllerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveControllerPartitions(ctx, r.pool)
}

func (r *tenantEngineRepository) CreateTenantWorkerPartition(ctx context.Context) (string, error) {
	partition, err := r.queries.CreateTenantWorkerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) DeleteTenantWorkerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteTenantWorkerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantEngineRepository) RebalanceAllTenantWorkerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllTenantWorkerPartitions(ctx, r.pool)
}

func (r *tenantEngineRepository) RebalanceInactiveTenantWorkerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveTenantWorkerPartitions(ctx, r.pool)
}

func (r *tenantEngineRepository) UpdateSchedulerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return "", err
	}

	defer sqlchelpers.DeferRollback(ctx, r.l, tx.Rollback)

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, "SET statement_timeout=5000")

	if err != nil {
		return "", err
	}

	partition, err := r.queries.SchedulerPartitionHeartbeat(ctx, tx, partitionId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// create a new partition
			partition, err = r.queries.CreateSchedulerPartition(ctx, tx, getPartitionName())

			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) ListTenantsBySchedulerPartition(ctx context.Context, schedulerPartitionId string) ([]*dbsqlc.Tenant, error) {
	if schedulerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsBySchedulerPartitionId(ctx, r.pool, schedulerPartitionId)
}

func (r *tenantEngineRepository) CreateSchedulerPartition(ctx context.Context) (string, error) {

	partition, err := r.queries.CreateSchedulerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantEngineRepository) DeleteSchedulerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteSchedulerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantEngineRepository) RebalanceAllSchedulerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllSchedulerPartitions(ctx, r.pool)
}

func (r *tenantEngineRepository) RebalanceInactiveSchedulerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveSchedulerPartitions(ctx, r.pool)
}

func getPartitionName() pgtype.Text {
	hostname, ok := os.LookupEnv("HOSTNAME")

	if !ok || hostname == "" {
		return pgtype.Text{}
	}

	return sqlchelpers.TextFromStr(hostname)
}
