package grpcoperator

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const defaultSlotCount = 100

// registerOpts carries the validation rules for a register request.
type registerOpts struct {
	Name string `validate:"required,hatchetName"`
}

// Register upserts the operator by name and creates the worker for this connection, or resumes
// the worker named by req.WorkerId when it still belongs to the operator. The worker starts with
// no actions: the client adds them on the Listen stream.
func (s *OperatorServiceImpl) Register(ctx context.Context, req *v1contracts.OperatorRegisterRequest) (*v1contracts.OperatorRegisterResponse, error) {
	tenant, ok := tenantFromContext(ctx)

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	if err := s.v.Validate(registerOpts{Name: req.Name}); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid register request: %s", err.Error())
	}

	op, err := s.operators.UpsertGRPCOperator(ctx, tenant.ID, req.Name)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not upsert GRPC operator %s", req.Name)
		return nil, err
	}

	workerId, resumed, err := s.resumeWorker(ctx, op, req)

	if err != nil {
		return nil, err
	}

	if !resumed {
		workerId, err = s.createWorker(ctx, tenant, op, req)

		if err != nil {
			return nil, err
		}
	}

	if len(req.Labels) > 0 {
		if err := s.upsertLabels(ctx, workerId, req.Labels); err != nil {
			return nil, err
		}
	}

	s.l.Info().Ctx(ctx).
		Str("operator_name", op.Name).
		Str("operator_id", op.ID.String()).
		Str("worker_id", workerId.String()).
		Bool("resumed", resumed).
		Msg("operator worker registered")

	return &v1contracts.OperatorRegisterResponse{
		TenantId:   tenant.ID.String(),
		OperatorId: op.ID.String(),
		WorkerId:   workerId.String(),
		Resumed:    resumed,
	}, nil
}

// resumeWorker returns the worker named by req.WorkerId when it belongs to op. It reports false
// when no worker id was sent or the worker no longer exists for this operator, in which case the
// caller creates a new one; a worker that exists but belongs to another operator is treated the
// same way rather than rejected, since the client only ever learns about its own workers.
func (s *OperatorServiceImpl) resumeWorker(ctx context.Context, op *sqlcv1.V1Operator, req *v1contracts.OperatorRegisterRequest) (uuid.UUID, bool, error) {
	if req.WorkerId == nil || *req.WorkerId == "" {
		return uuid.Nil, false, nil
	}

	workerId, err := uuid.Parse(*req.WorkerId)

	if err != nil {
		return uuid.Nil, false, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", *req.WorkerId)
	}

	worker, err := s.workers.GetWorkerForEngine(ctx, op.TenantID, workerId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.l.Debug().Ctx(ctx).Msgf("worker %s not found for GRPC operator %s, creating a new worker", workerId, op.ID)
			return uuid.Nil, false, nil
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s for GRPC operator %s", workerId, op.ID)
		return uuid.Nil, false, err
	}

	if worker.OperatorId == nil || *worker.OperatorId != op.ID {
		s.l.Debug().Ctx(ctx).Msgf("worker %s does not belong to GRPC operator %s, creating a new worker", workerId, op.ID)
		return uuid.Nil, false, nil
	}

	return workerId, true, nil
}

func (s *OperatorServiceImpl) createWorker(ctx context.Context, tenant *sqlcv1.Tenant, op *sqlcv1.V1Operator, req *v1contracts.OperatorRegisterRequest) (uuid.UUID, error) {
	slotConfig := req.SlotConfig

	if len(slotConfig) == 0 {
		slotConfig = map[string]int32{repository.SlotTypeDefault: defaultSlotCount}
	}

	operatorId := op.ID

	opts := &repository.CreateWorkerOpts{
		DispatcherId: s.dispatcherId,
		Name:         op.Name,
		SlotConfig:   slotConfig,
		OperatorId:   &operatorId,
	}

	if req.RuntimeInfo != nil {
		opts.RuntimeInfo = &repository.RuntimeInfo{
			SdkVersion:      req.RuntimeInfo.SdkVersion,
			Language:        req.RuntimeInfo.Language,
			LanguageVersion: req.RuntimeInfo.LanguageVersion,
			Os:              req.RuntimeInfo.Os,
			Extra:           req.RuntimeInfo.Extra,
		}
	}

	worker, err := s.workers.CreateNewWorker(ctx, tenant.ID, opts)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not create worker for GRPC operator %s", op.ID)
		return uuid.Nil, err
	}

	s.analytics.Count(ctx, analytics.Worker, analytics.Register, analytics.Props(
		"operator_kind", string(sqlcv1.V1OperatorKindGRPC),
		"operator_name", op.Name,
		"has_labels", len(req.Labels) > 0,
		"has_runtime_info", req.RuntimeInfo != nil,
		"has_slot_config", len(req.SlotConfig) > 0,
	))

	return worker.ID, nil
}

func (s *OperatorServiceImpl) upsertLabels(ctx context.Context, workerId uuid.UUID, labels map[string]*contracts.WorkerLabels) error {
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
