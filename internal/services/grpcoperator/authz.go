package grpcoperator

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// authorizeOperator resolves the calling operator from the hatchet-operator-id metadata and
// checks that it is a GRPC operator owned by the token's tenant. Lookups are cached for the
// cache's TTL; misses and errors are never cached, so a freshly created operator is visible on
// the next call.
func (s *OperatorServiceImpl) authorizeOperator(ctx context.Context) (*sqlcv1.V1Operator, error) {
	tenant, ok := tenantFromContext(ctx)

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing %s metadata", OperatorIdMetadataKey)
	}

	values := md.Get(OperatorIdMetadataKey)

	if len(values) == 0 || values[0] == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing %s metadata", OperatorIdMetadataKey)
	}

	operatorId, err := uuid.Parse(values[0])

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid %s metadata: %s is not a uuid", OperatorIdMetadataKey, values[0])
	}

	op, err := cache.MakeCacheable(s.cache, "grpc-operator:"+operatorId.String(), func() (*sqlcv1.V1Operator, error) {
		return s.operators.GetOperatorById(ctx, operatorId)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.PermissionDenied, "operator %s is not a GRPC operator for this tenant", operatorId)
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get operator %s", operatorId)
		return nil, err
	}

	if op.TenantID != tenant.ID || op.Kind != sqlcv1.V1OperatorKindGRPC {
		return nil, status.Errorf(codes.PermissionDenied, "operator %s is not a GRPC operator for this tenant", operatorId)
	}

	return op, nil
}

// authorizeOperatorWorker checks that workerIdStr names a worker owned by op. It is the guard
// on every RPC that carries a worker id, so an operator cannot act on another operator's
// worker. A worker that does not exist is reported the same way as one owned by someone else.
func (s *OperatorServiceImpl) authorizeOperatorWorker(ctx context.Context, op *sqlcv1.V1Operator, workerIdStr string) (*sqlcv1.GetWorkerForEngineRow, error) {
	workerId, err := uuid.Parse(workerIdStr)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", workerIdStr)
	}

	worker, err := s.workers.GetWorkerForEngine(ctx, op.TenantID, workerId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.PermissionDenied, "worker %s does not belong to operator %s", workerId, op.ID)
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s for operator %s", workerId, op.ID)
		return nil, err
	}

	if worker.OperatorId == nil || *worker.OperatorId != op.ID {
		return nil, status.Errorf(codes.PermissionDenied, "worker %s does not belong to operator %s", workerId, op.ID)
	}

	return worker, nil
}
