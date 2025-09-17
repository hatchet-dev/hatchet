package workers

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	transformersv1 "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkerService) WorkerList(ctx echo.Context, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return t.workerListV0(ctx, tenant, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return t.workerListV1(ctx, tenant, request)
	default:
		return nil, fmt.Errorf("unsupported tenant version: %s", string(tenant.Version))
	}
}

func (t *WorkerService) workerListV0(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	workers, err := t.config.APIRepository.Worker().ListWorkers(tenantId, &repository.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	})

	if err != nil {
		return nil, err
	}

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

func (t *WorkerService) workerListV1(ctx echo.Context, tenant *dbsqlc.Tenant, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	sixSecAgo := time.Now().Add(-24 * time.Hour)

	workers, err := t.config.V1.Workers().ListWorkers(tenantId, &repository.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	})

	if err != nil {
		return nil, err
	}

	workerIdSet := make(map[string]struct{})

	for _, worker := range workers {
		workerIdSet[sqlchelpers.UUIDToStr(worker.Worker.ID)] = struct{}{}
	}

	workerIds := make([]string, 0, len(workerIdSet))
	for workerId := range workerIdSet {
		workerIds = append(workerIds, workerId)
	}

	workerIdToActionIds, err := t.config.APIRepository.Worker().GetWorkerActionsByWorkerId(
		sqlchelpers.UUIDToStr(tenant.ID),
		workerIds,
	)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Worker, len(workers))

	for i, worker := range workers {
		workerCp := worker
		slots := int(worker.RemainingSlots)
		actions := workerIdToActionIds[sqlchelpers.UUIDToStr(workerCp.Worker.ID)]

		rows[i] = *transformersv1.ToWorkerSqlc(&workerCp.Worker, &slots, &workerCp.WebhookUrl.String, actions)
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
		},
	), nil
}
