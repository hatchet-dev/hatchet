package workers

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *WorkerService) WorkerList(ctx echo.Context, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
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
