package grpcoperator

import (
	"context"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// PauseWorker stops the scheduler assigning to one of the operator's workers, or lets it be
// assigned to again. It answers once the change is committed, so an operator that pauses before
// draining knows no further work will arrive, and it works while the Listen stream is being torn
// down or reconnected because it is its own call.
func (s *OperatorServiceImpl) PauseWorker(ctx context.Context, req *v1contracts.OperatorPauseWorkerRequest) (*v1contracts.OperatorPauseWorkerResponse, error) {
	tenant, op, err := s.authorizeOperator(ctx)

	if err != nil {
		return nil, err
	}

	worker, err := s.authorizeWorker(ctx, tenant, op, req.WorkerId)

	if err != nil {
		return nil, err
	}

	if err := s.svc.PauseWorker(ctx, tenant, worker.ID, req.Paused); err != nil {
		return nil, err
	}

	s.l.Info().Ctx(ctx).
		Str("operator_id", op.ID.String()).
		Str("worker_id", worker.ID.String()).
		Bool("paused", req.Paused).
		Msg("operator worker pause state changed")

	return &v1contracts.OperatorPauseWorkerResponse{
		WorkerId: worker.ID.String(),
		Paused:   req.Paused,
	}, nil
}
