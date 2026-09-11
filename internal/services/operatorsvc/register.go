package operatorsvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// defaultSlotCount is the slot count a worker gets when the caller asks for none.
const defaultSlotCount = 100

// RegisterOpts describes the operator to upsert, or the existing row to register under, and the
// worker to back this session.
type RegisterOpts struct {
	// OperatorId is an existing operator row, as claimed by the in-process claimer. Name and
	// Kind are taken from the row and no upsert happens; the row's worker_id is pointed at the
	// session's worker, which is how ClaimOperators recognises the assignment on later polls.
	OperatorId *uuid.UUID

	// Name is the operator name, unique per (tenant, kind). Ignored when OperatorId is set.
	Name string

	// Kind is how the operator is hosted. GRPC (an out-of-process operator) and SERVERLESS
	// (the serverless operator, in either mode) rows are upserted; the in-process kinds are
	// registered by OperatorId. Ignored when OperatorId is set.
	Kind sqlcv1.V1OperatorKind

	// WorkerName names the worker row. It defaults to the operator name, which is what one
	// worker per connection looks like in the dashboard.
	WorkerName string

	// SlotConfig maps slot type to max units, defaulting to {"default": 100}.
	SlotConfig map[string]int32

	Labels      map[string]*contracts.WorkerLabels
	RuntimeInfo *contracts.RuntimeInfo

	// ResumeWorkerId names a worker of this operator to reuse instead of creating one. A
	// worker that no longer exists, or that belongs to another operator, is replaced by a new
	// one rather than refused, since a caller only ever learns about its own workers.
	ResumeWorkerId *uuid.UUID
}

// Registration is the identity the engine assigned to the caller.
type Registration struct {
	Operator *sqlcv1.V1Operator

	TenantId   uuid.UUID
	OperatorId uuid.UUID
	WorkerId   uuid.UUID

	// Resumed reports whether WorkerId is the worker ResumeWorkerId named.
	Resumed bool
}

// registerNameOpts carries the validation rules for an operator name.
type registerNameOpts struct {
	Name string `validate:"required,hatchetName"`
}

// Register upserts the operator by (tenant, name, kind), or loads the row OperatorId names, and
// creates the worker for this session, or resumes the worker named by ResumeWorkerId when it
// still belongs to the operator. The worker starts with no actions: the caller links them on
// its session.
func (s *Service) Register(ctx context.Context, tenant *sqlcv1.Tenant, opts RegisterOpts) (Registration, error) {
	if tenant == nil {
		return Registration{}, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	var op *sqlcv1.V1Operator
	var err error

	if opts.OperatorId != nil {
		op, err = s.loadOperator(ctx, tenant, *opts.OperatorId)
	} else {
		if err := s.v.Validate(registerNameOpts{Name: opts.Name}); err != nil {
			return Registration{}, status.Errorf(codes.InvalidArgument, "invalid register request: %s", err.Error())
		}

		op, err = s.upsertOperator(ctx, tenant, opts)
	}

	if err != nil {
		return Registration{}, err
	}

	workerId, resumed, err := s.resumeWorker(ctx, op, opts.ResumeWorkerId)

	if err != nil {
		return Registration{}, err
	}

	if !resumed {
		workerId, err = s.createWorker(ctx, tenant, op, opts)

		if err != nil {
			return Registration{}, err
		}
	}

	if opts.OperatorId != nil {
		if err := s.pointOperatorAtWorker(ctx, op, workerId); err != nil {
			return Registration{}, err
		}
	}

	if len(opts.Labels) > 0 {
		if err := s.upsertLabels(ctx, workerId, opts.Labels); err != nil {
			return Registration{}, err
		}
	}

	s.l.Info().Ctx(ctx).
		Str("operator_name", op.Name).
		Str("operator_id", op.ID.String()).
		Str("worker_id", workerId.String()).
		Bool("resumed", resumed).
		Msg("operator worker registered")

	return Registration{
		Operator:   op,
		TenantId:   tenant.ID,
		OperatorId: op.ID,
		WorkerId:   workerId,
		Resumed:    resumed,
	}, nil
}

// loadOperator returns the existing row a claimed registration names, refusing a row of another
// tenant the way an unknown row is refused: the caller only ever learns about its own rows.
func (s *Service) loadOperator(ctx context.Context, tenant *sqlcv1.Tenant, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	op, err := s.operators.GetOperatorById(ctx, operatorId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "operator %s does not exist for this tenant", operatorId)
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get operator %s", operatorId)
		return nil, err
	}

	if op.TenantID != tenant.ID {
		return nil, status.Errorf(codes.NotFound, "operator %s does not exist for this tenant", operatorId)
	}

	return op, nil
}

// pointOperatorAtWorker records the session's worker on the operator row, so ClaimOperators
// sees the operator as assigned to this worker's dispatcher.
func (s *Service) pointOperatorAtWorker(ctx context.Context, op *sqlcv1.V1Operator, workerId uuid.UUID) error {
	if _, err := s.operators.UpdateOperator(ctx, op.TenantID, op.ID, repository.UpdateOperatorOpts{WorkerId: &workerId}); err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not point operator %s at worker %s", op.ID, workerId)
		return err
	}

	return nil
}

// upsertOperator upserts the row a named registration stands for. Each upsertable kind has its
// own statement because the rows are unique per (tenant, name, kind).
func (s *Service) upsertOperator(ctx context.Context, tenant *sqlcv1.Tenant, opts RegisterOpts) (*sqlcv1.V1Operator, error) {
	var op *sqlcv1.V1Operator
	var err error

	switch opts.Kind {
	case sqlcv1.V1OperatorKindGRPC:
		op, err = s.operators.UpsertGRPCOperator(ctx, tenant.ID, opts.Name)
	case sqlcv1.V1OperatorKindSERVERLESS:
		op, err = s.operators.UpsertServerlessOperator(ctx, tenant.ID, opts.Name)
	default:
		return nil, fmt.Errorf("operator kind %q cannot be registered through a session", opts.Kind)
	}

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not upsert %s operator %s", opts.Kind, opts.Name)
		return nil, err
	}

	return op, nil
}

// resumeWorker returns the worker named by resumeWorkerId when it belongs to op, and clears the
// pause a previous session left on it so an operator that was paused for draining comes back
// assignable. It reports false when no worker id was given or the worker no longer exists for
// this operator, in which case the caller creates a new one.
func (s *Service) resumeWorker(ctx context.Context, op *sqlcv1.V1Operator, resumeWorkerId *uuid.UUID) (uuid.UUID, bool, error) {
	if resumeWorkerId == nil {
		return uuid.Nil, false, nil
	}

	workerId := *resumeWorkerId

	worker, err := s.workers.GetWorkerForEngine(ctx, op.TenantID, workerId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.l.Debug().Ctx(ctx).Msgf("worker %s not found for operator %s, creating a new worker", workerId, op.ID)
			return uuid.Nil, false, nil
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s for operator %s", workerId, op.ID)
		return uuid.Nil, false, err
	}

	if worker.OperatorId == nil || *worker.OperatorId != op.ID {
		s.l.Debug().Ctx(ctx).Msgf("worker %s does not belong to operator %s, creating a new worker", workerId, op.ID)
		return uuid.Nil, false, nil
	}

	unpaused := false

	if _, err := s.workers.UpdateWorker(ctx, op.TenantID, workerId, &repository.UpdateWorkerOpts{IsPaused: &unpaused}); err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not clear the pause on resumed worker %s", workerId)
		return uuid.Nil, false, err
	}

	return workerId, true, nil
}

func (s *Service) createWorker(ctx context.Context, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator, opts RegisterOpts) (uuid.UUID, error) {
	slotConfig := opts.SlotConfig

	if len(slotConfig) == 0 {
		slotConfig = map[string]int32{repository.SlotTypeDefault: defaultSlotCount}
	}

	name := opts.WorkerName

	if name == "" {
		name = op.Name
	}

	operatorId := op.ID

	createOpts := &repository.CreateWorkerOpts{
		DispatcherId: s.dispatcherId,
		Name:         name,
		SlotConfig:   slotConfig,
		OperatorId:   &operatorId,
		OperatorKind: op.Kind,
	}

	if opts.RuntimeInfo != nil {
		createOpts.RuntimeInfo = &repository.RuntimeInfo{
			SdkVersion:      opts.RuntimeInfo.SdkVersion,
			Language:        opts.RuntimeInfo.Language,
			LanguageVersion: opts.RuntimeInfo.LanguageVersion,
			Os:              opts.RuntimeInfo.Os,
			Extra:           opts.RuntimeInfo.Extra,
		}
	}

	worker, err := s.workers.CreateNewWorker(ctx, tenant.ID, createOpts)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not create worker for operator %s", op.ID)
		return uuid.Nil, err
	}

	s.analytics.Count(ctx, analytics.Worker, analytics.Register, analytics.Props(
		"operator_kind", string(op.Kind),
		"operator_name", op.Name,
		"has_labels", len(opts.Labels) > 0,
		"has_runtime_info", opts.RuntimeInfo != nil,
		"has_slot_config", len(opts.SlotConfig) > 0,
	))

	return worker.ID, nil
}

func (s *Service) upsertLabels(ctx context.Context, workerId uuid.UUID, labels map[string]*contracts.WorkerLabels) error {
	affinities := make([]repository.UpsertWorkerLabelOpts, 0, len(labels))

	for key, config := range labels {
		if err := s.v.Validate(config); err != nil {
			return status.Errorf(codes.InvalidArgument, "Invalid affinity config: %s", err.Error())
		}

		affinities = append(affinities, repository.UpsertWorkerLabelOpts{
			Key:      key,
			IntValue: config.IntValue,
			StrValue: config.StrValue,
		})
	}

	if _, err := s.workers.UpsertWorkerLabels(ctx, workerId, affinities); err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not upsert worker labels for worker %s", workerId)
		return err
	}

	return nil
}
