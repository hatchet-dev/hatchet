package workers

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (t *WorkerService) WorkerList(ctx echo.Context, request gen.WorkerListRequestObject) (gen.WorkerListResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	sixSecAgo := time.Now().Add(-6 * time.Second)

	workers, err := t.config.Repository.Worker().ListWorkers(tenant.ID, &repository.ListWorkersOpts{
		LastHeartbeatAfter: &sixSecAgo,
	})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Worker, len(workers))

	for i, worker := range workers {
		workerCp := worker
		rows[i] = *transformers.ToWorkerSqlc(&workerCp.Worker)
	}

	return gen.WorkerList200JSONResponse(
		gen.WorkerList{
			Rows: &rows,
		},
	), nil
}
