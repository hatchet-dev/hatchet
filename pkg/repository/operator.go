package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type OperatorRepository interface {
	CreateOperator(ctx context.Context, tenantId uuid.UUID, opts CreateOperatorOpts) (*sqlcv1.V1Operator, error)
	GetOperatorById(ctx context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error)
	ListOperators(ctx context.Context, tenantId uuid.UUID, opts ListOperatorsOpts) ([]*sqlcv1.V1Operator, int64, error)
	UpdateOperator(ctx context.Context, tenantId, operatorId uuid.UUID, opts UpdateOperatorOpts) (*sqlcv1.V1Operator, error)
	DeleteOperator(ctx context.Context, tenantId, operatorId uuid.UUID) (*sqlcv1.V1Operator, error)

	// ClaimOperators returns all operators which should be run by this dispatcher (unassigned,
	// on an inactive dispatcher, or already assigned to this dispatcher). It does not create
	// workers; the claimer registers one per claimed row through the operator service, which
	// points the row's worker_id at it so later polls see the assignment.
	ClaimOperators(ctx context.Context, dispatcherId uuid.UUID) ([]*sqlcv1.V1Operator, error)

	// UpsertOperator registers an operator by (tenant, name, kind), returning the existing row
	// on repeat registrations with its leasing set to the one given. A SELF row never gets a
	// worker_id on the operator row: each registration owns its own worker, created through
	// WorkerRepository.CreateNewWorker with CreateWorkerOpts.OperatorId and linked via
	// "Worker"."operatorId".
	UpsertOperator(ctx context.Context, tenantId uuid.UUID, opts UpsertOperatorOpts) (*sqlcv1.V1Operator, error)

	// ListDAGOrchestrationActions returns the orchestration action IDs ("{name}_orchestrator")
	// for all DAG workflows of a tenant. The DAG operator polls this to keep its registered
	// actions in sync with the tenant's DAGs.
	ListDAGOrchestrationActions(ctx context.Context, tenantId uuid.UUID) ([]string, error)

	// HasDAGOperator reports whether any DAG operator is registered for the given tenant.
	HasDAGOperator(ctx context.Context, tenantId uuid.UUID) (bool, error)

	// CountEvictedDAGOrchestratorRuns returns the number of DAG orchestrator runs for the
	// tenant that are currently evicted while waiting on durable events. They hold no worker
	// slots, so this is the only way to see how many runs the DAG operator has in flight.
	CountEvictedDAGOrchestratorRuns(ctx context.Context, tenantId uuid.UUID) (int64, error)
}

type operatorRepository struct {
	*sharedRepository
}

func newOperatorRepository(shared *sharedRepository) OperatorRepository {
	return &operatorRepository{
		sharedRepository: shared,
	}
}

type CreateOperatorOpts struct {
	Name string                `json:"name" validate:"required"`
	Kind sqlcv1.V1OperatorKind `json:"kind" validate:"required,oneof=DAG GRPC"`

	// Leasing is who keeps the operator alive: MANAGED rows are claimed by a dispatcher and
	// built from a factory, SELF rows register their own workers.
	Leasing sqlcv1.V1OperatorLeasing `json:"leasing" validate:"required,oneof=MANAGED SELF"`
	Config  []byte                   `json:"config" validate:"required"`
}

func (r *operatorRepository) CreateOperator(ctx context.Context, tenantId uuid.UUID, opts CreateOperatorOpts) (*sqlcv1.V1Operator, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.CreateOperator(ctx, r.pool, sqlcv1.CreateOperatorParams{
		Tenantid: tenantId,
		Name:     opts.Name,
		Kind:     opts.Kind,
		Leasing:  opts.Leasing,
		Config:   opts.Config,
	})
}

type UpsertOperatorOpts struct {
	Name string                `validate:"required"`
	Kind sqlcv1.V1OperatorKind `validate:"required,oneof=DAG GRPC"`

	// Leasing is what the row is set to whether it is created or found.
	Leasing sqlcv1.V1OperatorLeasing `validate:"required,oneof=MANAGED SELF"`
}

func (r *operatorRepository) UpsertOperator(ctx context.Context, tenantId uuid.UUID, opts UpsertOperatorOpts) (*sqlcv1.V1Operator, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.UpsertOperator(ctx, r.pool, sqlcv1.UpsertOperatorParams{
		Tenantid: tenantId,
		Name:     opts.Name,
		Kind:     opts.Kind,
		Leasing:  opts.Leasing,
	})
}

func (r *operatorRepository) GetOperatorById(ctx context.Context, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	return r.queries.GetOperator(ctx, r.pool, operatorId)
}

type ListOperatorsOpts struct {
	// Kind optionally filters to a single operator kind.
	Kind   *sqlcv1.V1OperatorKind `json:"kind"`
	Limit  int64                  `json:"limit" validate:"omitnil,min=1"`
	Offset int64                  `json:"offset" validate:"omitnil,min=0"`
}

func (r *operatorRepository) ListOperators(ctx context.Context, tenantId uuid.UUID, opts ListOperatorsOpts) ([]*sqlcv1.V1Operator, int64, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	kind := nullOperatorKind(opts.Kind)

	operators, err := r.queries.ListOperators(ctx, r.pool, sqlcv1.ListOperatorsParams{
		Tenantid:       tenantId,
		Kind:           kind,
		Operatorlimit:  opts.Limit,
		Operatoroffset: opts.Offset,
	})

	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountOperators(ctx, r.pool, sqlcv1.CountOperatorsParams{
		Tenantid: tenantId,
		Kind:     kind,
	})

	if err != nil {
		return nil, 0, err
	}

	return operators, count, nil
}

type UpdateOperatorOpts struct {
	Name   *string `json:"name"`
	Config []byte  `json:"config"`

	// WorkerId points the operator row at the worker that runs it, which is how ClaimOperators
	// recognises an operator as assigned to that worker's dispatcher.
	WorkerId *uuid.UUID `json:"-"`
}

func (r *operatorRepository) UpdateOperator(ctx context.Context, tenantId, operatorId uuid.UUID, opts UpdateOperatorOpts) (*sqlcv1.V1Operator, error) {
	params := sqlcv1.UpdateOperatorParams{
		Tenantid: tenantId,
		ID:       operatorId,
		Config:   opts.Config,
		WorkerId: opts.WorkerId,
	}

	if opts.Name != nil {
		params.Name = pgtype.Text{
			String: *opts.Name,
			Valid:  true,
		}
	}

	return r.queries.UpdateOperator(ctx, r.pool, params)
}

func (r *operatorRepository) DeleteOperator(ctx context.Context, tenantId, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	return r.queries.DeleteOperator(ctx, r.pool, sqlcv1.DeleteOperatorParams{
		Tenantid: tenantId,
		ID:       operatorId,
	})
}

// nullOperatorKind maps an optional kind filter to the nullable sqlc type.
func nullOperatorKind(kind *sqlcv1.V1OperatorKind) sqlcv1.NullV1OperatorKind {
	if kind == nil {
		return sqlcv1.NullV1OperatorKind{}
	}

	return sqlcv1.NullV1OperatorKind{
		V1OperatorKind: *kind,
		Valid:          true,
	}
}

func (r *operatorRepository) ClaimOperators(ctx context.Context, dispatcherId uuid.UUID) ([]*sqlcv1.V1Operator, error) {
	return r.queries.ClaimOperators(ctx, r.pool, dispatcherId)
}

func (r *operatorRepository) ListDAGOrchestrationActions(ctx context.Context, tenantId uuid.UUID) ([]string, error) {
	return r.queries.ListDAGOrchestrationActionsForTenant(ctx, r.pool, tenantId)
}

func (r *operatorRepository) HasDAGOperator(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	return r.hasDAGOperator(ctx, tenantId)
}

func (r *operatorRepository) CountEvictedDAGOrchestratorRuns(ctx context.Context, tenantId uuid.UUID) (int64, error) {
	return r.queries.CountEvictedDAGOrchestratorRuns(ctx, r.pool, tenantId)
}
