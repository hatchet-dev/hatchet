package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (t *WorkerService) WorkerList(ctx echo.Context, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	reqCtx, span := telemetry.NewSpan(ctx.Request().Context(), "GET /api/v1/tenants/{tenant}/worker")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
		telemetry.AttributeKV{Key: "tenant.version", Value: string(tenant.Version)},
	)

	ctx.SetRequest(ctx.Request().WithContext(reqCtx))

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return t.workerListV0(ctx, reqCtx, tenant, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return t.workerListV1(ctx, reqCtx, tenant, request)
	default:
		err := fmt.Errorf("unsupported tenant version: %s", string(tenant.Version))
		span.RecordError(err)
		return nil, err
	}
}

func (t *WorkerService) workerListV0(ctx echo.Context, reqCtx context.Context, tenant *dbsqlc.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	opts := &repository.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	}

	_, listSpan := telemetry.NewSpan(reqCtx, "worker-service.v0.list-workers")
	defer listSpan.End()

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
	)

	workers, err := t.config.APIRepository.Worker().ListWorkers(tenantId, opts)

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

func (t *WorkerService) workerListV1(ctx echo.Context, reqCtx context.Context, tenant *dbsqlc.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	opts := &repository.ListWorkersOpts{
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

	workerIdSet := make(map[string]struct{})

	for _, worker := range workers {
		workerIdSet[sqlchelpers.UUIDToStr(worker.Worker.ID)] = struct{}{}
	}

	workerIds := make([]string, 0, len(workerIdSet))
	for workerId := range workerIdSet {
		workerIds = append(workerIds, workerId)
	}

	_, actionsSpan := telemetry.NewSpan(listCtx, "worker-service.v1.get-worker-actions")
	defer actionsSpan.End()

	telemetry.WithAttributes(actionsSpan,
		telemetry.AttributeKV{Key: "workers.unique_ids.count", Value: len(workerIds)},
	)

	workerIdToActionIds, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(tenant.ID),
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
		actions := workerIdToActionIds[sqlchelpers.UUIDToStr(workerCp.Worker.ID)]

		rows[i] = *transformersv1.ToWorkerSqlc(&workerCp.Worker, &slots, &workerCp.WebhookUrl.String, actions, nil)
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
		},
	), nil
}
