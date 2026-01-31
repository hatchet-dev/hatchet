package workers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (t *WorkerService) WorkerList(ctx echo.Context, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	switch tenant.Version {
	case sqlcv1.TenantMajorEngineVersionV0:
		return t.workerListV0(ctx, tenant, request)
	case sqlcv1.TenantMajorEngineVersionV1:
		return t.workerListV1(ctx, tenant, request)
	default:
		err := fmt.Errorf("unsupported tenant version: %s", string(tenant.Version))
		return nil, err
	}
}

func (t *WorkerService) workerListV0(ctx echo.Context, tenant *sqlcv1.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	reqCtx := ctx.Request().Context()
	tenantId := tenant.ID

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	opts := &v1.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	}

	_, listSpan := telemetry.NewSpan(reqCtx, "worker-service.v0.list-workers")
	defer listSpan.End()

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
	)

	workers, err := t.config.V1.Workers().ListWorkers(tenantId, opts)

	if err != nil {
		listSpan.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "workers.count", Value: len(workers)},
	)

	rows := make([]gen.Worker, len(workers))

	for i, worker := range workers {
		workerCp := worker
		slots := int(worker.RemainingSlots)

		rows[i] = *transformers.ToWorkerSqlc(&workerCp.Worker, &slots, &workerCp.WebhookUrl.String, nil)
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
		},
	), nil
}

func (t *WorkerService) workerListV1(ctx echo.Context, tenant *sqlcv1.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	reqCtx := ctx.Request().Context()
	tenantId := tenant.ID

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	opts := &v1.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	}

	listCtx, listSpan := telemetry.NewSpan(reqCtx, "worker-service.v1.list-workers")
	defer listSpan.End()

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
	)

	workers, err := t.config.V1.Workers().ListWorkers(tenantId, opts)

	if err != nil {
		listSpan.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "workers.count", Value: len(workers)},
	)

	workerIdSet := make(map[uuid.UUID]struct{})

	for _, worker := range workers {
		workerIdSet[worker.Worker.ID] = struct{}{}
	}

	workerIds := make([]uuid.UUID, 0, len(workerIdSet))
	for workerId := range workerIdSet {
		workerIds = append(workerIds, workerId)
	}

	_, actionsSpan := telemetry.NewSpan(listCtx, "worker-service.v1.get-worker-actions")
	defer actionsSpan.End()

	telemetry.WithAttributes(actionsSpan,
		telemetry.AttributeKV{Key: "workers.unique_ids.count", Value: len(workerIds)},
	)

	workerIdToActionIds, err := t.config.V1.Workers().GetWorkerActionsByWorkerId(
		tenant.ID,
		workerIds,
	)

	if err != nil {
		actionsSpan.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(actionsSpan,
		telemetry.AttributeKV{Key: "worker_actions.mappings.count", Value: len(workerIdToActionIds)},
	)

	rows := make([]gen.Worker, len(workers))

	for i, worker := range workers {
		workerCp := worker
		slots := int(worker.RemainingSlots)
		actions := workerIdToActionIds[workerCp.Worker.ID.String()]

		rows[i] = *transformersv1.ToWorkerSqlc(&workerCp.Worker, &slots, &workerCp.WebhookUrl.String, actions, nil)
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
		},
	), nil
}
