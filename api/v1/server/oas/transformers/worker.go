package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func ToWorker(worker *db.WorkerModel) *gen.Worker {
	res := &gen.Worker{
		Metadata: *toAPIMetadata(worker.ID, worker.CreatedAt, worker.UpdatedAt),
		Name:     worker.Name,
	}

	if lastHeartbeatAt, ok := worker.LastHeartbeatAt(); ok {
		res.LastHeartbeatAt = &lastHeartbeatAt
	}

	if worker.RelationsWorker.Actions != nil {
		if actions := worker.Actions(); actions != nil {
			apiActions := make([]string, len(actions))

			for i, action := range actions {
				apiActions[i] = action.ID
			}

			res.Actions = &apiActions
		}
	}

	return res
}
