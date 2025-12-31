package postgres

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
)

type stepRunAPIRepository struct {
	*sharedRepository
}

func NewStepRunAPIRepository(shared *sharedRepository) repository.StepRunAPIRepository {
	return &stepRunAPIRepository{
		sharedRepository: shared,
	}
}

type stepRunEngineRepository struct {
	*sharedRepository

	cf *server.ConfigFileRuntime

	queueActionTenantCache *cache.Cache

	updateConcurrentFactor int
	maxHashFactor          int
}

func NewStepRunEngineRepository(shared *sharedRepository, cf *server.ConfigFileRuntime, rlCache *cache.Cache, queueCache *cache.Cache) *stepRunEngineRepository {
	return &stepRunEngineRepository{
		sharedRepository:       shared,
		cf:                     cf,
		updateConcurrentFactor: cf.UpdateConcurrentFactor,
		maxHashFactor:          cf.UpdateHashFactor,
		queueActionTenantCache: queueCache,
	}
}
