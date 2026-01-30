package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateTenantOpts struct {
	// (required) the tenant name
	Name string `validate:"required"`

	// (required) the tenant slug
	Slug string `validate:"required,hatchetName"`

	// (optional) the tenant ID
	ID *string `validate:"omitempty,uuid"`

	// (optional) the tenant data retention period
	DataRetentionPeriod *string `validate:"omitempty,duration"`

	// (optional) the tenant engine version
	EngineVersion *sqlcv1.TenantMajorEngineVersion `validate:"omitempty"`

	// (optional) the tenant environment type
	Environment *string `validate:"omitempty,oneof=local development production"`

	// (optional) additional onboarding data
	OnboardingData map[string]interface{}
}

type UpdateTenantOpts struct {
	Name *string

	AnalyticsOptOut *bool `validate:"omitempty"`

	AlertMemberEmails *bool `validate:"omitempty"`

	Version *sqlcv1.NullTenantMajorEngineVersion `validate:"omitempty"`
}

type CreateTenantMemberOpts struct {
	Role   string `validate:"required,oneof=OWNER ADMIN MEMBER"`
	UserId string `validate:"required,uuid"`
}

type UpdateTenantMemberOpts struct {
	Role *string `validate:"omitempty,oneof=OWNER ADMIN MEMBER"`
}

type GetQueueMetricsOpts struct {
	// (optional) a list of workflow ids to filter by
	WorkflowIds []string `validate:"omitempty,dive,uuid"`

	// (optional) exact metadata to filter by
	AdditionalMetadata map[string]interface{} `validate:"omitempty"`
}

type QueueMetric struct {
	// the total number of PENDING_ASSIGNMENT step runs in the queue
	PendingAssignment int `json:"pending_assignment"`

	// the total number of PENDING step runs in the queue
	Pending int `json:"pending"`

	// the total number of RUNNING step runs in the queue
	Running int `json:"running"`
}

type GetQueueMetricsResponse struct {
	Total QueueMetric `json:"total"`

	ByWorkflowId map[string]QueueMetric `json:"by_workflow"`
}

type TenantRepository interface {
	RegisterCreateCallback(callback UnscopedCallback[*sqlcv1.Tenant])

	// CreateTenant creates a new tenant.
	CreateTenant(ctx context.Context, opts *CreateTenantOpts) (*sqlcv1.Tenant, error)

	// UpdateTenant updates an existing tenant in the db.
	UpdateTenant(ctx context.Context, tenantId string, opts *UpdateTenantOpts) (*sqlcv1.Tenant, error)

	// GetTenantByID returns the tenant with the given id
	GetTenantByID(ctx context.Context, tenantId string) (*sqlcv1.Tenant, error)

	// GetTenantBySlug returns the tenant with the given slug
	GetTenantBySlug(ctx context.Context, slug string) (*sqlcv1.Tenant, error)

	// CreateTenantMember creates a new member in the tenant
	CreateTenantMember(ctx context.Context, tenantId string, opts *CreateTenantMemberOpts) (*sqlcv1.PopulateTenantMembersRow, error)

	// GetTenantMemberByID returns the tenant member with the given id
	GetTenantMemberByID(ctx context.Context, memberId string) (*sqlcv1.PopulateTenantMembersRow, error)

	// GetTenantMemberByUserID returns the tenant member with the given user id
	GetTenantMemberByUserID(ctx context.Context, tenantId string, userId string) (*sqlcv1.PopulateTenantMembersRow, error)

	// GetTenantMemberByEmail returns the tenant member with the given email
	GetTenantMemberByEmail(ctx context.Context, tenantId string, email string) (*sqlcv1.PopulateTenantMembersRow, error)

	// ListTenantMembers returns the list of tenant members for the given tenant
	ListTenantMembers(ctx context.Context, tenantId string) ([]*sqlcv1.PopulateTenantMembersRow, error)

	// UpdateTenantMember updates the tenant member with the given id
	UpdateTenantMember(ctx context.Context, memberId string, opts *UpdateTenantMemberOpts) (*sqlcv1.PopulateTenantMembersRow, error)

	// DeleteTenantMember deletes the tenant member with the given id
	DeleteTenantMember(ctx context.Context, memberId string) error

	// GetQueueMetrics returns the queue metrics for the given tenant
	GetQueueMetrics(ctx context.Context, tenantId string, opts *GetQueueMetricsOpts) (*GetQueueMetricsResponse, error)

	// ListTenants lists all tenants in the instance
	ListTenants(ctx context.Context) ([]*sqlcv1.Tenant, error)

	// Gets the tenant corresponding to the "internal" tenant if it's assigned to this controller.
	// Returns nil if the tenant is not assigned to this controller.
	GetInternalTenantForController(ctx context.Context, controllerPartitionId string) (*sqlcv1.Tenant, error)

	// ListTenantsByPartition lists all tenants in the given partition
	ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error)

	ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error)

	ListTenantsBySchedulerPartition(ctx context.Context, schedulerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error)

	// CreateEnginePartition creates a new partition for tenants within the engine
	CreateControllerPartition(ctx context.Context) (string, error)

	// UpdateControllerPartitionHeartbeat updates the heartbeat for the given partition. If the partition no longer exists,
	// it creates a new partition and returns the new partition id. Otherwise, it returns the existing partition id.
	UpdateControllerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteControllerPartition(ctx context.Context, id string) error

	RebalanceAllControllerPartitions(ctx context.Context) error

	RebalanceInactiveControllerPartitions(ctx context.Context) error

	CreateSchedulerPartition(ctx context.Context) (string, error)

	UpdateSchedulerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteSchedulerPartition(ctx context.Context, id string) error

	RebalanceAllSchedulerPartitions(ctx context.Context) error

	RebalanceInactiveSchedulerPartitions(ctx context.Context) error

	CreateTenantWorkerPartition(ctx context.Context) (string, error)

	UpdateWorkerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error)

	DeleteTenantWorkerPartition(ctx context.Context, id string) error

	RebalanceAllTenantWorkerPartitions(ctx context.Context) error

	RebalanceInactiveTenantWorkerPartitions(ctx context.Context) error

	DeleteTenant(ctx context.Context, id string) error

	GetTenantUsageData(ctx context.Context, tenantId string) (*sqlcv1.GetTenantUsageDataRow, error)
}

type tenantRepository struct {
	*sharedRepository

	cache                cache.Cacheable
	defaultTenantVersion sqlcv1.TenantMajorEngineVersion
	createCallbacks      []UnscopedCallback[*sqlcv1.Tenant]
}

func newTenantRepository(shared *sharedRepository, cacheDuration time.Duration) TenantRepository {
	return &tenantRepository{
		sharedRepository:     shared,
		cache:                cache.New(cacheDuration),
		defaultTenantVersion: sqlcv1.TenantMajorEngineVersionV1,
	}
}

func (r *tenantRepository) RegisterCreateCallback(callback UnscopedCallback[*sqlcv1.Tenant]) {
	if r.createCallbacks == nil {
		r.createCallbacks = make([]UnscopedCallback[*sqlcv1.Tenant], 0)
	}

	r.createCallbacks = append(r.createCallbacks, callback)
}

func (r *tenantRepository) CreateTenant(ctx context.Context, opts *CreateTenantOpts) (*sqlcv1.Tenant, error) {
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

	engineVersion := r.defaultTenantVersion
	if opts.EngineVersion != nil {
		engineVersion = *opts.EngineVersion
	}

	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	var environment sqlcv1.NullTenantEnvironment
	if opts.Environment != nil {
		environment = sqlcv1.NullTenantEnvironment{
			TenantEnvironment: sqlcv1.TenantEnvironment(*opts.Environment),
			Valid:             true,
		}
	} else {
		// Default to development environment if none is specified
		environment = sqlcv1.NullTenantEnvironment{
			TenantEnvironment: sqlcv1.TenantEnvironmentDevelopment,
			Valid:             true,
		}
	}

	var onboardingData []byte
	if opts.OnboardingData != nil {
		onboardingData, err = json.Marshal(opts.OnboardingData)
		if err != nil {
			return nil, err
		}
	}

	createTenant, err := r.queries.CreateTenant(context.Background(), tx, sqlcv1.CreateTenantParams{
		ID:                  uuid.MustParse(tenantId),
		Slug:                opts.Slug,
		Name:                opts.Name,
		DataRetentionPeriod: dataRetentionPeriod,
		Version: sqlcv1.NullTenantMajorEngineVersion{
			TenantMajorEngineVersion: engineVersion,
			Valid:                    true,
		},
		OnboardingData: onboardingData,
		Environment:    environment,
	})

	if err != nil {
		return nil, err
	}

	_, err = r.queries.CreateTenantAlertingSettings(ctx, tx, uuid.MustParse(tenantId))

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	for _, cb := range r.createCallbacks {
		cb.Do(r.l, createTenant)
	}

	return createTenant, nil
}

func (r *tenantRepository) UpdateTenant(ctx context.Context, id string, opts *UpdateTenantOpts) (*sqlcv1.Tenant, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateTenantParams{
		ID: uuid.MustParse(id),
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

	if opts.Version != nil && opts.Version.Valid {
		params.Version = *opts.Version
	}

	return r.queries.UpdateTenant(
		ctx,
		r.pool,
		params,
	)
}

func (r *tenantRepository) GetTenantByID(ctx context.Context, id string) (*sqlcv1.Tenant, error) {
	return cache.MakeCacheable(r.cache, "api"+id, func() (*sqlcv1.Tenant, error) {
		return r.queries.GetTenantByID(ctx, r.pool, uuid.MustParse(id))
	})
}

func (r *tenantRepository) GetTenantBySlug(ctx context.Context, slug string) (*sqlcv1.Tenant, error) {
	return r.queries.GetTenantBySlug(
		ctx,
		r.pool,
		slug,
	)
}

func (r *tenantRepository) CreateTenantMember(ctx context.Context, tenantId string, opts *CreateTenantMemberOpts) (*sqlcv1.PopulateTenantMembersRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createdMember, err := r.queries.CreateTenantMember(
		ctx,
		r.pool,
		sqlcv1.CreateTenantMemberParams{
			Tenantid: uuid.MustParse(tenantId),
			Userid:   uuid.MustParse(opts.UserId),
			Role:     sqlcv1.TenantMemberRole(opts.Role),
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, createdMember.ID)
}

func (r *tenantRepository) GetTenantMemberByID(ctx context.Context, memberId string) (*sqlcv1.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByID(
		ctx,
		r.pool,
		uuid.MustParse(memberId),
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantRepository) GetTenantMemberByUserID(ctx context.Context, tenantId string, userId string) (*sqlcv1.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByUserID(
		ctx,
		r.pool,
		sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: uuid.MustParse(tenantId),
			Userid:   uuid.MustParse(userId),
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantRepository) ListTenantMembers(ctx context.Context, tenantId string) ([]*sqlcv1.PopulateTenantMembersRow, error) {
	members, err := r.queries.ListTenantMembers(
		ctx,
		r.pool,
		uuid.MustParse(tenantId),
	)

	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(members))

	for i, member := range members {
		ids[i] = member.ID
	}

	return r.populateTenantMembers(ctx, ids)
}

func (r *tenantRepository) GetTenantMemberByEmail(ctx context.Context, tenantId string, email string) (*sqlcv1.PopulateTenantMembersRow, error) {
	member, err := r.queries.GetTenantMemberByEmail(
		ctx,
		r.pool,
		sqlcv1.GetTenantMemberByEmailParams{
			Tenantid: uuid.MustParse(tenantId),
			Email:    email,
		},
	)

	if err != nil {
		return nil, err
	}

	return r.populateSingleTenantMember(ctx, member.ID)
}

func (r *tenantRepository) UpdateTenantMember(ctx context.Context, memberId string, opts *UpdateTenantMemberOpts) (*sqlcv1.PopulateTenantMembersRow, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateTenantMemberParams{
		ID: uuid.MustParse(memberId),
	}

	if opts.Role != nil {
		params.Role = sqlcv1.NullTenantMemberRole{
			TenantMemberRole: sqlcv1.TenantMemberRole(*opts.Role),
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

func (r *sharedRepository) populateSingleTenantMember(ctx context.Context, ids uuid.UUID) (*sqlcv1.PopulateTenantMembersRow, error) {
	res, err := r.populateTenantMembers(ctx, []uuid.UUID{ids})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, pgx.ErrNoRows
	}

	return res[0], nil
}

func (r *sharedRepository) populateTenantMembers(ctx context.Context, ids []uuid.UUID) ([]*sqlcv1.PopulateTenantMembersRow, error) {
	return r.queries.PopulateTenantMembers(
		ctx,
		r.pool,
		ids,
	)
}

func (r *tenantRepository) DeleteTenantMember(ctx context.Context, memberId string) error {
	return r.queries.DeleteTenantMember(
		ctx,
		r.pool,
		uuid.MustParse(memberId),
	)
}

func (r *tenantRepository) GetQueueMetrics(ctx context.Context, tenantId string, opts *GetQueueMetricsOpts) (*GetQueueMetricsResponse, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	totalParams := sqlcv1.GetTenantTotalQueueMetricsParams{
		TenantId: uuid.MustParse(tenantId),
	}

	workflowParams := sqlcv1.GetTenantWorkflowQueueMetricsParams{
		TenantId: uuid.MustParse(tenantId),
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
		uuids := make([]uuid.UUID, len(opts.WorkflowIds))

		for i, id := range opts.WorkflowIds {
			uuids[i] = uuid.MustParse(id)
		}

		workflowParams.WorkflowIds = uuids
		totalParams.WorkflowIds = uuids
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

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

	workflowMetricsMap := make(map[string]QueueMetric)

	for _, metric := range workflowMetrics {
		workflowMetricsMap[metric.WorkflowId.String()] = QueueMetric{
			PendingAssignment: int(metric.PendingAssignmentCount),
			Pending:           int(metric.PendingCount),
			Running:           int(metric.RunningCount),
		}
	}

	return &GetQueueMetricsResponse{
		Total: QueueMetric{
			PendingAssignment: int(total.PendingAssignmentCount),
			Pending:           int(total.PendingCount),
			Running:           int(total.RunningCount),
		},
		ByWorkflowId: workflowMetricsMap,
	}, nil
}

func (r *tenantRepository) ListTenants(ctx context.Context) ([]*sqlcv1.Tenant, error) {
	return r.queries.ListTenants(ctx, r.pool)
}

func (r *tenantRepository) UpdateControllerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
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

func (r *tenantRepository) UpdateWorkerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
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

func (r *tenantRepository) GetInternalTenantForController(ctx context.Context, controllerPartitionId string) (*sqlcv1.Tenant, error) {
	tenant, err := r.queries.GetInternalTenantForController(ctx, r.pool, controllerPartitionId)

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (r *tenantRepository) ListTenantsByControllerPartition(ctx context.Context, controllerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error) {
	if controllerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByControllerPartitionId(ctx, r.pool, sqlcv1.ListTenantsByControllerPartitionIdParams{
		ControllerPartitionId: controllerPartitionId,
		Majorversion:          majorVersion,
	})
}

func (r *tenantRepository) ListTenantsByWorkerPartition(ctx context.Context, workerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error) {
	if workerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsByTenantWorkerPartitionId(ctx, r.pool, sqlcv1.ListTenantsByTenantWorkerPartitionIdParams{
		WorkerPartitionId: workerPartitionId,
		Majorversion:      majorVersion,
	})
}

func (r *tenantRepository) CreateControllerPartition(ctx context.Context) (string, error) {

	partition, err := r.queries.CreateControllerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantRepository) DeleteControllerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteControllerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantRepository) RebalanceAllControllerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllControllerPartitions(ctx, r.pool)
}

func (r *tenantRepository) RebalanceInactiveControllerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveControllerPartitions(ctx, r.pool)
}

func (r *tenantRepository) CreateTenantWorkerPartition(ctx context.Context) (string, error) {
	partition, err := r.queries.CreateTenantWorkerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantRepository) DeleteTenantWorkerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteTenantWorkerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantRepository) RebalanceAllTenantWorkerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllTenantWorkerPartitions(ctx, r.pool)
}

func (r *tenantRepository) RebalanceInactiveTenantWorkerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveTenantWorkerPartitions(ctx, r.pool)
}

func (r *tenantRepository) UpdateSchedulerPartitionHeartbeat(ctx context.Context, partitionId string) (string, error) {
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

func (r *tenantRepository) ListTenantsBySchedulerPartition(ctx context.Context, schedulerPartitionId string, majorVersion sqlcv1.TenantMajorEngineVersion) ([]*sqlcv1.Tenant, error) {
	if schedulerPartitionId == "" {
		return nil, fmt.Errorf("partitionId is required")
	}

	return r.queries.ListTenantsBySchedulerPartitionId(ctx, r.pool, sqlcv1.ListTenantsBySchedulerPartitionIdParams{
		SchedulerPartitionId: schedulerPartitionId,
		Majorversion:         majorVersion,
	})
}

func (r *tenantRepository) CreateSchedulerPartition(ctx context.Context) (string, error) {

	partition, err := r.queries.CreateSchedulerPartition(ctx, r.pool, getPartitionName())

	if err != nil {
		return "", err
	}

	return partition.ID, nil
}

func (r *tenantRepository) DeleteSchedulerPartition(ctx context.Context, id string) error {
	_, err := r.queries.DeleteSchedulerPartition(ctx, r.pool, id)
	return err
}

func (r *tenantRepository) RebalanceAllSchedulerPartitions(ctx context.Context) error {
	return r.queries.RebalanceAllSchedulerPartitions(ctx, r.pool)
}

func (r *tenantRepository) RebalanceInactiveSchedulerPartitions(ctx context.Context) error {
	return r.queries.RebalanceInactiveSchedulerPartitions(ctx, r.pool)
}

func (r *tenantRepository) DeleteTenant(ctx context.Context, id string) error {
	return r.queries.DeleteTenant(ctx, r.pool, uuid.MustParse(id))
}

func (r *tenantRepository) GetTenantUsageData(ctx context.Context, tenantId string) (*sqlcv1.GetTenantUsageDataRow, error) {
	return r.queries.GetTenantUsageData(ctx, r.pool, uuid.MustParse(tenantId))
}

func getPartitionName() pgtype.Text {
	hostname, ok := os.LookupEnv("HOSTNAME")

	if !ok || hostname == "" {
		return pgtype.Text{}
	}

	return sqlchelpers.TextFromStr(hostname)
}
