package postgres

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
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantAPIRepository struct {
	*sharedRepository

	cache cache.Cacheable
}

func NewTenantAPIRepository(shared *sharedRepository, cache cache.Cacheable) repository.TenantAPIRepository {
	return &tenantAPIRepository{
		sharedRepository: shared,
		cache:            cache,
	}
}

func (r *tenantAPIRepository) CreateTenant(ctx context.Context, opts *repository.CreateTenantOpts) (*dbsqlc.Tenant, error) {
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

	tx, err := r.pool.Begin(ctx)

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

	_, err = r.queries.CreateTenantAlertingSettings(ctx, tx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return createTenant, nil
}

func (r *tenantAPIRepository) UpdateTenant(ctx context.Context, id string, opts *repository.UpdateTenantOpts) (*dbsqlc.Tenant, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateTenantParams{
		ID: sqlchelpers.UUIDFromStr(id),
	}

	if opts.Name != nil {
		params.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	if opts.AnalyticsOptOut != nil {
		params.AnalyticsOptOut = sqlchelpers.BoolFromBoolean(*opts.AnalyticsOptOut)
	}

	if opts.AlertMemberEmails != nil {
		params.AlertMemberEmails = sqlchelpers.BoolFromBoolean(*opts.AlertMemberEmails)
	}

	return r.queries.UpdateTenant(
		ctx,
		r.pool,
		params,
	)
}

func (r *tenantAPIRepository) GetTenantByID(ctx context.Context, id string) (*dbsqlc.Tenant, error) {
	return cache.MakeCacheable(r.cache, "api"+id, func() (*dbsqlc.Tenant, error) {
		return r.queries.GetTenantByID(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
	})
}

func (r *tenantAPIRepository) GetTenantBySlug(ctx context.Context, slug string) (*dbsqlc.Tenant, error) {
	return r.queries.GetTenantBySlug(
		ctx,
		r.pool,
		slug,
	)
}

func (r *tenantAPIRepository) CreateTenantMember(ctx context.Context, tenantId string, opts *repository.CreateTenantMemberOpts) (*dbsqlc.PopulateTenantMembersRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createdMember, err := r.queries.CreateTenantMember(
		ctx,
		r.pool,
		dbsqlc.CreateTenantMemberParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Userid:   sqlchelpers.UUIDFromStr(opts.UserId),
			Role:     dbsqlc.TenantMemberRole(opts.Role),
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, createdMember.ID)
}

func (r *tenantAPIRepository) GetTenantMemberByID(ctx context.Context, memberId string) (*dbsqlc.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByID(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(memberId),
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantAPIRepository) GetTenantMemberByUserID(ctx context.Context, tenantId string, userId string) (*dbsqlc.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByUserID(
		ctx,
		r.pool,
		dbsqlc.GetTenantMemberByUserIDParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Userid:   sqlchelpers.UUIDFromStr(userId),
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantAPIRepository) ListTenantMembers(ctx context.Context, tenantId string) ([]*dbsqlc.PopulateTenantMembersRow, error) {
	members, err := r.queries.ListTenantMembers(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(tenantId),
	)

	if err != nil {
		return nil, err
	}

	ids := make([]pgtype.UUID, len(members))

	for i, member := range members {
		ids[i] = member.ID
	}

	return r.populateTenantMembers(ctx, ids)
}

func (r *tenantAPIRepository) GetTenantMemberByEmail(ctx context.Context, tenantId string, email string) (*dbsqlc.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByEmail(
		ctx,
		r.pool,
		dbsqlc.GetTenantMemberByEmailParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Email:    email,
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantAPIRepository) UpdateTenantMember(ctx context.Context, memberId string, opts *repository.UpdateTenantMemberOpts) (*dbsqlc.PopulateTenantMembersRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateTenantMemberParams{
		ID: sqlchelpers.UUIDFromStr(memberId),
	}

	if opts.Role != nil {
		params.Role = dbsqlc.NullTenantMemberRole{
			TenantMemberRole: dbsqlc.TenantMemberRole(*opts.Role),
			Valid:            true,
		}
	}

	updatedMember, err := r.queries.UpdateTenantMember(
		ctx,
		r.pool,
		params,
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, updatedMember.ID)
}

func (r *sharedRepository) populateSingleTenantMember(ctx context.Context, ids pgtype.UUID) (*dbsqlc.PopulateTenantMembersRow, error) {
	res, err := r.populateTenantMembers(ctx, []pgtype.UUID{ids})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, pgx.ErrNoRows
	}

	return res[0], nil
}

func (r *sharedRepository) populateTenantMembers(ctx context.Context, ids []pgtype.UUID) ([]*dbsqlc.PopulateTenantMembersRow, error) {
	return r.queries.PopulateTenantMembers(
		ctx,
		r.pool,
		ids,
	)
}

func (r *tenantAPIRepository) DeleteTenantMember(ctx context.Context, memberId string) error {
	return r.queries.DeleteTenantMember(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(memberId),
	)
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
	return cache.MakeCacheable(r.cache, tenantId, func() (*dbsqlc.Tenant, error) {
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

func (r *tenantEngineRepository) GetInternalTenantForController(ctx context.Context, controllerPartitionId string) (*dbsqlc.Tenant, error) {
	tenant, err := r.queries.GetInternalTenantForController(ctx, r.pool, controllerPartitionId)

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (r *tenantEngineRepository) ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error) {
	if controllerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByControllerPartitionId(ctx, r.pool, dbsqlc.ListTenantsByControllerPartitionIdParams{
		ControllerPartitionId: controllerPartitionId,
		Majorversion:          majorVersion,
	})
}

func (r *tenantEngineRepository) ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error) {
	if workerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByTenantWorkerPartitionId(ctx, r.pool, dbsqlc.ListTenantsByTenantWorkerPartitionIdParams{
		WorkerPartitionId: workerPartitionId,
		Majorversion:      majorVersion,
	})
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

func (r *tenantEngineRepository) ListTenantsBySchedulerPartition(ctx context.Context, schedulerPartitionId string, majorVersion dbsqlc.TenantMajorEngineVersion) ([]*dbsqlc.Tenant, error) {
	if schedulerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsBySchedulerPartitionId(ctx, r.pool, dbsqlc.ListTenantsBySchedulerPartitionIdParams{
		SchedulerPartitionId: schedulerPartitionId,
		Majorversion:         majorVersion,
	})
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
