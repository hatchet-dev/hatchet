package transformers

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func ToWorkerLabels(labels []*dbsqlc.ListWorkerLabelsRow) *[]gen.WorkerLabel {
	resp := make([]gen.WorkerLabel, len(labels))

	for i := range labels {
		// c := gen.WorkerAffinityComparator(labels[i].Comparator)
		// w := int(labels[i].Weight)

		var value *string
		if v, ok := labels[i].Value.(string); ok {
			value = &v
		} else if v, ok := labels[i].Value.(*string); ok {
			value = v
		} else {
			// Handle other types or set a default value
			// For example, you can log an error or set value to nil
			log.Println("Type assertion failed for Value")
			value = nil
		}

		id := fmt.Sprintf("%d", labels[i].ID)

		resp[i] = gen.WorkerLabel{
			Metadata: *toAPIMetadata(id, labels[i].CreatedAt.Time, labels[i].UpdatedAt.Time),
			Key:      labels[i].Key,
			Value:    value,
		}
	}

	return &resp
}

func ToWorker(worker *db.WorkerModel) *gen.Worker {

	var dispatcherUuid uuid.UUID

	if id, ok := worker.DispatcherID(); ok {
		dispatcherUuid = uuid.MustParse(id)
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
		MaxRuns:      &worker.MaxRuns,
	}

	if lastHeartbeatAt, ok := worker.LastHeartbeatAt(); ok {
		res.LastHeartbeatAt = &lastHeartbeatAt
	}

	if lastListenerEstablished, ok := worker.LastListenerEstablished(); ok {
		res.LastListenerEstablished = &lastListenerEstablished
	}

	numSlots := 0
	for _, slot := range worker.Slots() {
		if _, ok := slot.StepRunID(); !ok {
			numSlots++
		}
	}
	res.AvailableRuns = &numSlots

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

	maxRuns := int(worker.MaxRuns)

	status := gen.ACTIVE

	if worker.LastHeartbeatAt.Time.Add(5 * time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	availableRuns := maxRuns - *slots

	res := &gen.Worker{
		Metadata:      *toAPIMetadata(pgUUIDToStr(worker.ID), worker.CreatedAt.Time, worker.UpdatedAt.Time),
		Name:          worker.Name,
		Status:        &status,
		DispatcherId:  &dispatcherId,
		MaxRuns:       &maxRuns,
		AvailableRuns: &availableRuns,
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	return res
}
