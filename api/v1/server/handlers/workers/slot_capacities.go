package workers

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

type slotAvailabilityRepository interface {
	ListWorkerSlotConfigs(tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int32, error)
	ListAvailableSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID, slotType string) (map[uuid.UUID]int32, error)
}

func buildWorkerSlotConfig(ctx context.Context, repo slotAvailabilityRepository, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]gen.WorkerSlotConfig, error) {
	if len(workerIds) == 0 {
		return map[uuid.UUID]map[string]gen.WorkerSlotConfig{}, nil
	}

	slotConfigByWorker, err := repo.ListWorkerSlotConfigs(tenantId, workerIds)
	if err != nil {
		return nil, fmt.Errorf("could not list worker slot config: %w", err)
	}

	slotTypes := make(map[string]struct{})
	for _, config := range slotConfigByWorker {
		for slotType := range config {
			slotTypes[slotType] = struct{}{}
		}
	}

	availableBySlotType := make(map[string]map[uuid.UUID]int32, len(slotTypes))
	for slotType := range slotTypes {
		available, err := repo.ListAvailableSlotsForWorkers(ctx, tenantId, workerIds, slotType)
		if err != nil {
			return nil, fmt.Errorf("could not list available slots for slot type %s: %w", slotType, err)
		}
		availableBySlotType[slotType] = available
	}

	result := make(map[uuid.UUID]map[string]gen.WorkerSlotConfig, len(slotConfigByWorker))
	for workerId, config := range slotConfigByWorker {
		workerSlots := make(map[string]gen.WorkerSlotConfig, len(config))
		for slotType, limit := range config {
			available := 0
			if slotAvailability, ok := availableBySlotType[slotType]; ok {
				if value, ok := slotAvailability[workerId]; ok {
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
