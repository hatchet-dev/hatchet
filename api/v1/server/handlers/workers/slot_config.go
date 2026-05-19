package workers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

type slotAvailabilityRepository interface {
	ListWorkerSlotConfigs(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotType string) (map[uuid.UUID]int32, error)
	ListAvailableSlotsForWorkersAndTypes(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotTypes []string) (map[uuid.UUID]map[string]int32, error)
}

func buildWorkerSlotConfig(ctx context.Context, repo slotAvailabilityRepository, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]gen.WorkerSlotConfig, error) {
	if len(workerIds) == 0 {
		return map[uuid.UUID]map[string]gen.WorkerSlotConfig{}, nil
	}

	slotConfigByWorker, err := repo.ListWorkerSlotConfigs(ctx, tenantId, workerIds)
	if err != nil {
		return nil, fmt.Errorf("could not list worker slot config: %w", err)
	}

	slotTypes := make(map[string]struct{})
	slotTypesArr := make([]string, 0)
	for _, config := range slotConfigByWorker {
		for slotType := range config {
			if _, ok := slotTypes[slotType]; ok {
				continue
			}

			slotTypes[slotType] = struct{}{}
			slotTypesArr = append(slotTypesArr, slotType)
		}
	}

	availableByWorker, err := repo.ListAvailableSlotsForWorkersAndTypes(ctx, tenantId, workerIds, slotTypesArr)
	if err != nil {
		return nil, fmt.Errorf("could not list available slots for workers and types: %w", err)
	}

	result := make(map[uuid.UUID]map[string]gen.WorkerSlotConfig, len(slotConfigByWorker))
	for workerId, config := range slotConfigByWorker {
		workerSlots := make(map[string]gen.WorkerSlotConfig, len(config))
		for slotType, limit := range config {
			available := 0
			if workerAvailability, ok := availableByWorker[workerId]; ok {
				if value, ok := workerAvailability[slotType]; ok {
					available = int(value)
				}
			}

			workerSlots[slotType] = gen.WorkerSlotConfig{
				Available: &available,
				Limit:     int(limit),
			}
		}
		result[workerId] = workerSlots
	}

	return result, nil
}
