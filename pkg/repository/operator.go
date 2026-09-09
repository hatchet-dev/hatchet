package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
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

	// CreateOperatorWorker creates a new worker for a single operator instance and points the
	// operator at it. It is called once per operator instantiation, separately from
	// ClaimOperators, so each running instance of an operator gets its own worker. slotConfig
	// (slot_type -> max units) is provided by the caller and may vary between operators.
	CreateOperatorWorker(ctx context.Context, dispatcherId uuid.UUID, operator *sqlcv1.V1Operator, slotConfig map[string]int32) (*sqlcv1.Worker, error)

	// UpdateOperatorWorkerActions links actions to the worker corresponding to the operator. Links
	// the worker already holds are kept, and the worker's action hash is recomputed from the links
	// the call leaves behind, so it is always the canonical digest of the linked set. The worker
	// must belong to the tenant.
	UpdateOperatorWorkerActions(ctx context.Context, tenantId, workerId uuid.UUID, actions []string) error

	// UpsertGRPCOperator registers an out-of-process operator by (tenant, name) with kind GRPC,
	// returning the existing row on repeat connects. GRPC operators never get a worker_id on the
	// operator row: each connection owns its own worker, created through
	// WorkerRepository.CreateNewWorker with CreateWorkerOpts.OperatorId and linked via
	// "Worker"."operatorId".
	UpsertGRPCOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error)

	// UpsertServerlessOperator registers the serverless operator by (tenant, name) with kind
	// SERVERLESS, returning the existing row on repeat registrations. Like GRPC operators the
	// row never gets a worker_id: each serverless registration owns its own worker, created
	// through WorkerRepository.CreateNewWorker with CreateWorkerOpts.OperatorId.
	UpsertServerlessOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error)

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
	Name   string                `json:"name" validate:"required"`
	Kind   sqlcv1.V1OperatorKind `json:"kind" validate:"required"`
	Config []byte                `json:"config" validate:"required"`
}

func (r *operatorRepository) CreateOperator(ctx context.Context, tenantId uuid.UUID, opts CreateOperatorOpts) (*sqlcv1.V1Operator, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.CreateOperator(ctx, r.pool, sqlcv1.CreateOperatorParams{
		Tenantid: tenantId,
		Name:     opts.Name,
		Kind:     opts.Kind,
		Config:   opts.Config,
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

func (r *operatorRepository) CreateOperatorWorker(ctx context.Context, dispatcherId uuid.UUID, operator *sqlcv1.V1Operator, slotConfig map[string]int32) (*sqlcv1.Worker, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	worker, err := r.queries.CreateOperatorWorker(ctx, tx, sqlcv1.CreateOperatorWorkerParams{
		Tenantid:     operator.TenantID,
		Name:         operator.Name,
		Dispatcherid: dispatcherId,
		Actionhash:   hashActions([]string{}),
		Operatorid:   operator.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create operator worker: %w", err)
	}

	// Create the worker's slot config (slot_type -> max units), the same way CreateNewWorker
	// does for regular workers.
	slotTypes := make([]string, 0, len(slotConfig))
	maxUnits := make([]int32, 0, len(slotConfig))

	for slotType, units := range slotConfig {
		slotTypes = append(slotTypes, slotType)
		maxUnits = append(maxUnits, units)
	}

	if len(slotTypes) > 0 {
		err = r.queries.CreateWorkerSlotConfigs(ctx, tx, sqlcv1.CreateWorkerSlotConfigsParams{
			Tenantid:  operator.TenantID,
			Workerid:  worker.ID,
			Slottypes: slotTypes,
			Maxunits:  maxUnits,
		})

		if err != nil {
			return nil, fmt.Errorf("could not create operator worker slot config: %w", err)
		}
	}

	// Point the operator at its new worker so ClaimOperators recognizes it as assigned to
	// this dispatcher on subsequent polls.
	_, err = r.queries.UpdateOperator(ctx, tx, sqlcv1.UpdateOperatorParams{
		ID:       operator.ID,
		Tenantid: operator.TenantID,
		WorkerId: &worker.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("could not point operator at new worker: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit operator worker creation: %w", err)
	}

	return worker, nil
}

func (r *operatorRepository) UpdateOperatorWorkerActions(ctx context.Context, tenantId, workerId uuid.UUID, actions []string) error {
	actions = dedupeActionIds(actions)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return err
	}

	defer rollback()

	// the row lock is held until commit, so this call and the worker repository's deltas for
	// the same worker apply one after the other and each recomputes the hash from the links it
	// leaves behind; the tenant is part of the lock's predicate
	if _, err := r.queries.LockWorkerActionHash(ctx, tx, sqlcv1.LockWorkerActionHashParams{
		Workerid: workerId,
		Tenantid: tenantId,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("worker %s does not belong to tenant %s: %w", workerId, tenantId, err)
		}

		return fmt.Errorf("could not lock worker %s: %w", workerId, err)
	}

	actionUUIDs := make([]uuid.UUID, len(actions))

	for i, action := range actions {
		dbAction, upsertErr := r.queries.UpsertAction(ctx, tx, sqlcv1.UpsertActionParams{
			Action:   action,
			Tenantid: tenantId,
		})

		if upsertErr != nil {
			return fmt.Errorf("could not upsert action: %w", upsertErr)
		}

		actionUUIDs[i] = dbAction.ID
	}

	if _, err := r.queries.LinkActionsToWorkerReturning(ctx, tx, sqlcv1.LinkActionsToWorkerReturningParams{
		Workerid:  workerId,
		Tenantid:  tenantId,
		Actionids: actionUUIDs,
	}); err != nil {
		return fmt.Errorf("could not link actions to worker: %w", err)
	}

	// the hash is the digest of every link the worker holds, not of the list this call was
	// given, so a call that repeats or extends an existing set stores the same value a delta
	// path would
	hash, err := r.queries.ComputeWorkerActionHash(ctx, tx, workerId)

	if err != nil {
		return fmt.Errorf("could not compute worker actions hash: %w", err)
	}

	if err := r.queries.UpdateWorkerActionsHash(ctx, tx, sqlcv1.UpdateWorkerActionsHashParams{
		Workerid:   workerId,
		Actionhash: hash,
	}); err != nil {
		return fmt.Errorf("could not update worker actions hash: %w", err)
	}

	return commit(ctx)
}

func (r *operatorRepository) UpsertGRPCOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error) {
	return r.queries.UpsertGRPCOperator(ctx, r.pool, sqlcv1.UpsertGRPCOperatorParams{
		Tenantid: tenantId,
		Name:     name,
	})
}

func (r *operatorRepository) UpsertServerlessOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error) {
	return r.queries.UpsertServerlessOperator(ctx, r.pool, sqlcv1.UpsertServerlessOperatorParams{
		Tenantid: tenantId,
		Name:     name,
	})
}
