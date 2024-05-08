package transformers

import (
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

func ToWorker(worker *db.WorkerModel) *gen.Worker {

	var dispatcherUuid uuid.UUID

	if id, ok := worker.DispatcherID(); ok {
		dispatcherUuid = uuid.MustParse(id)
	}

	var maxRuns int

	if runs, ok := worker.MaxRuns(); ok {
		maxRuns = runs
	}

	status := gen.ACTIVE

	if lastHeartbeat, ok := worker.LastHeartbeatAt(); ok && lastHeartbeat.Add(4*time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	res := &gen.Worker{
		Metadata:     *toAPIMetadata(worker.ID, worker.CreatedAt, worker.UpdatedAt),
		Name:         worker.Name,
		DispatcherId: &dispatcherUuid,
		Status:       &status,
		MaxRuns:      &maxRuns,
	}

	if lastHeartbeatAt, ok := worker.LastHeartbeatAt(); ok {
		res.LastHeartbeatAt = &lastHeartbeatAt
	}

	if semaphore, ok := worker.Semaphore(); ok {
		res.AvailableRuns = &semaphore.Slots
	}

	if worker.RelationsWorker.Actions != nil {
		if actions := worker.Actions(); actions != nil {
			apiActions := make([]string, len(actions))

			for i, action := range actions {
				apiActions[i] = action.ActionID
			}

			res.Actions = &apiActions
		}
	}

	return res
}

func ToWorkerSqlc(worker *dbsqlc.Worker, stepCount *int64, slots *int) *gen.Worker {

	dispatcherId := uuid.MustParse(pgUUIDToStr(worker.DispatcherId))

	maxRuns := int(worker.MaxRuns.Int32)

	status := gen.ACTIVE

	if worker.LastHeartbeatAt.Time.Add(4 * time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	res := &gen.Worker{
		Metadata:      *toAPIMetadata(pgUUIDToStr(worker.ID), worker.CreatedAt.Time, worker.UpdatedAt.Time),
		Name:          worker.Name,
		Status:        &status,
		DispatcherId:  &dispatcherId,
		MaxRuns:       &maxRuns,
		AvailableRuns: slots,
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	return res
}
