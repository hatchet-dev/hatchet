package workers

import (
	"fmt"
	"math"
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

	limit := 50
	offset := 0

	opts := &v1.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
		Limit:              &limit,
		Offset:             &offset,
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		opts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		opts.Offset = &offset
	}

	if request.Params.Statuses != nil {
		statuses := make([]string, len(*request.Params.Statuses))
		for i, s := range *request.Params.Statuses {
			statuses[i] = string(s)
		}
		opts.Statuses = statuses
	}

	_, listSpan := telemetry.NewSpan(reqCtx, "worker-service.v0.list-workers")
	defer listSpan.End()

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
	)

	workers, count, err := t.config.V1.Workers().ListWorkers(reqCtx, tenantId, opts)

	if err != nil {
		listSpan.RecordError(err)
		return nil, err
	}

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "workers.count", Value: len(workers)},
	)

	rows := make([]gen.Worker, len(workers))
	workerIds := make([]uuid.UUID, 0, len(workers))
	for _, worker := range workers {
		workerIds = append(workerIds, worker.Worker.ID)
	}

	workerSlotConfig, err := buildWorkerSlotConfig(reqCtx, t.config.V1.Workers(), tenantId, workerIds)
	if err != nil {
		listSpan.RecordError(err)
		return nil, err
	}

	for i, worker := range workers {
		workerCp := worker
		slotConfig := workerSlotConfig[workerCp.Worker.ID]
		rows[i] = *transformers.ToWorkerSqlc(&workerCp.Worker, slotConfig, nil)
	}

	totalPages := int64(math.Ceil(float64(count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}

func (t *WorkerService) workerListV1(ctx echo.Context, tenant *sqlcv1.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	reqCtx := ctx.Request().Context()
	tenantId := tenant.ID

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	limit := 10000 // default to 10k for backwards compat (more or less) to before when we had no `LIMIT` set
	offset := 0

	opts := &v1.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
		Limit:              &limit,
		Offset:             &offset,
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		opts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		opts.Offset = &offset
	}

	if request.Params.Statuses != nil {
		statuses := make([]string, len(*request.Params.Statuses))
		for i, s := range *request.Params.Statuses {
			statuses[i] = string(s)
		}
		opts.Statuses = statuses
	}

	listCtx, listSpan := telemetry.NewSpan(reqCtx, "worker-service.v1.list-workers")
	defer listSpan.End()

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "tenant.id", Value: tenant.ID},
	)

	workerRows, count, err := t.config.V1.Workers().ListWorkers(listCtx, tenantId, opts)

	if err != nil {
		listSpan.RecordError(err)
		return nil, err
	}

	workers := make([]sqlcv1.Worker, len(workerRows))

	for i, workerRow := range workerRows {
		workers[i] = workerRow.Worker
	}

	telemetry.WithAttributes(listSpan,
		telemetry.AttributeKV{Key: "workers.count", Value: len(workers)},
	)

	workerIdSet := make(map[uuid.UUID]struct{})

	for _, worker := range workers {
		workerIdSet[worker.ID] = struct{}{}
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

	workerIdToActionIds, err := t.config.V1.Workers().GetWorkerActionsForWorkers(
		listCtx,
		tenant.ID,
		workers,
	)

	if err != nil {
		actionsSpan.RecordError(err)
		return nil, err
	}

	workerIdToLabels, err := t.config.V1.Workers().ListWorkerLabels(listCtx, tenantId, workerIds)

	if err != nil {
		actionsSpan.RecordError(err)
		return nil, err
	}

	workerSlotConfig, err := buildWorkerSlotConfig(listCtx, t.config.V1.Workers(), tenant.ID, workerIds)
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
		actions := workerIdToActionIds[workerCp.ID.String()]
		slotConfig := workerSlotConfig[workerCp.ID]
		labels := workerIdToLabels[workerCp.ID]

		rows[i] = *transformersv1.ToWorkerSqlc(&workerCp, slotConfig, actions, nil, labels)
	}

	totalPages := int64(math.Ceil(float64(count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				CurrentPage: &currPage,
				NextPage:    &nextPage,
			},
		},
	), nil
}
