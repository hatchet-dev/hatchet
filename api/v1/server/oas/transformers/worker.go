package transformers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToWorkerLabels(labels []*sqlcv1.ListWorkerLabelsRow) *[]gen.WorkerLabel {
	resp := make([]gen.WorkerLabel, len(labels))

	for i := range labels {

		var value *string

		switch {
		case labels[i].IntValue.Valid:
			intValue := labels[i].IntValue.Int32
			stringValue := fmt.Sprintf("%d", intValue)
			value = &stringValue
		case labels[i].StrValue.Valid:
			value = &labels[i].StrValue.String
		default:
			value = nil
		}

		resp[i] = gen.WorkerLabel{
			// fixme: `id` needs to be a uuid
			Metadata: *toAPIMetadata(uuid.Nil, labels[i].CreatedAt.Time, labels[i].UpdatedAt.Time),
			Key:      labels[i].Key,
			Value:    value,
		}
	}

	return &resp
}

func ToWorkerRuntimeInfo(worker *sqlcv1.Worker) *gen.WorkerRuntimeInfo {

	runtime := &gen.WorkerRuntimeInfo{
		SdkVersion:      &worker.SdkVersion.String,
		LanguageVersion: &worker.LanguageVersion.String,
		Os:              &worker.Os.String,
		RuntimeExtra:    &worker.RuntimeExtra.String,
	}

	if worker.Language.Valid {
		lang := gen.WorkerRuntimeSDKs(worker.Language.WorkerSDKS)
		runtime.Language = &lang
	}

	return runtime
}

func ToWorkerSqlc(worker *sqlcv1.Worker, slotConfig map[string]gen.WorkerSlotConfig, actions []string) *gen.Worker {

	dispatcherId := worker.DispatcherId

	status := gen.ACTIVE

	if worker.IsPaused {
		status = gen.PAUSED
	}

	if worker.LastHeartbeatAt.Time.Add(5 * time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	var slotConfigInt *map[string]gen.WorkerSlotConfig
	if len(slotConfig) > 0 {
		tmp := make(map[string]gen.WorkerSlotConfig, len(slotConfig))
		for k, v := range slotConfig {
			tmp[k] = v
		}
		slotConfigInt = &tmp
	}

	res := &gen.Worker{
		Metadata:       *toAPIMetadata(worker.ID, worker.CreatedAt.Time, worker.UpdatedAt.Time),
		Name:           worker.Name,
		Type:           gen.WorkerType(worker.Type),
		Status:         &status,
		DispatcherId:   dispatcherId,
		SlotConfig:     slotConfigInt,
		RuntimeInfo:    ToWorkerRuntimeInfo(worker),
		WebhookId:      worker.WebhookId,
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	res.Actions = &actions

	return res
}
