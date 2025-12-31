package postgres

import (
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type jobRunAPIRepository struct {
	*sharedRepository
}

func NewJobRunAPIRepository(shared *sharedRepository) repository.JobRunAPIRepository {
	return &jobRunAPIRepository{
		sharedRepository: shared,
	}
}

type jobRunEngineRepository struct {
	*sharedRepository
}

func NewJobRunEngineRepository(shared *sharedRepository) repository.JobRunEngineRepository {

	return &jobRunEngineRepository{
		sharedRepository: shared,
	}
}
