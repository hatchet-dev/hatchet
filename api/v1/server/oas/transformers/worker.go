package transformers

import (
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

	res := &gen.Worker{
		Metadata:     *toAPIMetadata(worker.ID, worker.CreatedAt, worker.UpdatedAt),
		Name:         worker.Name,
		DispatcherId: &dispatcherUuid,
		Status:       (*gen.WorkerStatus)(&worker.Status),
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

func ToWorkerSqlc(worker *dbsqlc.Worker, stepCount *int64) *gen.Worker {

	dispatcherId := uuid.MustParse(pgUUIDToStr(worker.DispatcherId))

	maxRuns := int(worker.MaxRuns.Int32)
	availableRuns := maxRuns - int(*stepCount)

	res := &gen.Worker{
		Metadata:      *toAPIMetadata(pgUUIDToStr(worker.ID), worker.CreatedAt.Time, worker.UpdatedAt.Time),
		Name:          worker.Name,
		Status:        (*gen.WorkerStatus)(&worker.Status),
		DispatcherId:  &dispatcherId,
		MaxRuns:       &maxRuns,
		AvailableRuns: &availableRuns,
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	return res
}
