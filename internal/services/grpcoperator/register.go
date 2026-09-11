package grpcoperator

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Register upserts the operator by name and creates the worker for this connection, or resumes
// the worker named by req.WorkerId when it still belongs to the operator. The worker starts with
// no actions: the client adds them on the Listen stream.
func (s *OperatorServiceImpl) Register(ctx context.Context, req *v1contracts.OperatorRegisterRequest) (*v1contracts.OperatorRegisterResponse, error) {
	tenant, ok := tenantFromContext(ctx)

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	resumeWorkerId, err := parseResumeWorkerId(req.WorkerId)

	if err != nil {
		return nil, err
	}

	// A wire registration is a contract operator that keeps itself alive through this
	// connection's Listen stream, so the row is GRPC and SELF; its workers are metered.
	reg, err := s.svc.Register(ctx, tenant, operatorsvc.RegisterOpts{
		Name:           req.Name,
		Kind:           sqlcv1.V1OperatorKindGRPC,
		Leasing:        sqlcv1.V1OperatorLeasingSELF,
		SlotConfig:     req.SlotConfig,
		Labels:         req.Labels,
		RuntimeInfo:    req.RuntimeInfo,
		ResumeWorkerId: resumeWorkerId,
	})

	if err != nil {
		return nil, err
	}

	return &v1contracts.OperatorRegisterResponse{
		TenantId:   reg.TenantId.String(),
		OperatorId: reg.OperatorId.String(),
		WorkerId:   reg.WorkerId.String(),
		Resumed:    reg.Resumed,
	}, nil
}

// parseResumeWorkerId reads the optional worker id a client sends to resume its previous worker.
// An absent or empty value asks for a new worker.
func parseResumeWorkerId(raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}

	workerId, err := uuid.Parse(*raw)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", *raw)
	}

	return &workerId, nil
}
