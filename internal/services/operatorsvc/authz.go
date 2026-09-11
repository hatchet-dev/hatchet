package operatorsvc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// AuthorizeOperator resolves operatorId and checks that it is a self-leased GRPC operator owned
// by tenant. Lookups are cached for the cache's TTL; misses and errors are never cached, so a
// freshly created operator is visible on the next call. Only such operators are reachable from
// outside the engine, which is the only caller that authorizes today: a DAG row is the
// engine's own, and a row the engine leases is driven by the claimer, never over the wire.
func (s *Service) AuthorizeOperator(ctx context.Context, tenant *sqlcv1.Tenant, operatorId uuid.UUID) (*sqlcv1.V1Operator, error) {
	if tenant == nil {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	op, err := cache.MakeCacheable(s.cache, "operator:"+operatorId.String(), func() (*sqlcv1.V1Operator, error) {
		return s.operators.GetOperatorById(ctx, operatorId)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.PermissionDenied, "operator %s is not a self-leased GRPC operator for this tenant", operatorId)
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get operator %s", operatorId)
		return nil, err
	}

	if op.TenantID != tenant.ID || op.Kind != sqlcv1.V1OperatorKindGRPC || op.Leasing != sqlcv1.V1OperatorLeasingSELF {
		return nil, status.Errorf(codes.PermissionDenied, "operator %s is not a self-leased GRPC operator for this tenant", operatorId)
	}

	return op, nil
}

// AuthorizeWorker checks that workerId names a worker owned by the operator. It is the guard on
// every call that carries a worker id, so an operator cannot act on another operator's worker.
// A worker that does not exist is reported the same way as one owned by someone else.
func (s *Service) AuthorizeWorker(ctx context.Context, tenant *sqlcv1.Tenant, operatorId uuid.UUID, workerId uuid.UUID) (*sqlcv1.GetWorkerForEngineRow, error) {
	if tenant == nil {
		return nil, status.Error(codes.Unauthenticated, "tenant not found in request context")
	}

	worker, err := s.workers.GetWorkerForEngine(ctx, tenant.ID, workerId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.PermissionDenied, "worker %s does not belong to operator %s", workerId, operatorId)
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s for operator %s", workerId, operatorId)
		return nil, err
	}

	if worker.OperatorId == nil || *worker.OperatorId != operatorId {
		return nil, status.Errorf(codes.PermissionDenied, "worker %s does not belong to operator %s", workerId, operatorId)
	}

	return worker, nil
}
