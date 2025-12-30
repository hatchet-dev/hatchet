package postgres

import (
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type eventAPIRepository struct {
	*sharedRepository
}

func NewEventAPIRepository(shared *sharedRepository) repository.EventAPIRepository {
	return &eventAPIRepository{
		sharedRepository: shared,
	}
}

type eventEngineRepository struct {
	*sharedRepository

	m                   *metered.Metered
	callbacks           []repository.TenantScopedCallback[*dbsqlc.Event]
	createEventKeyCache *lru.Cache[string, bool]
}

func NewEventEngineRepository(shared *sharedRepository, m *metered.Metered, bufferConf buffer.ConfigFileBuffer) repository.EventEngineRepository {
	createEventKeyCache, _ := lru.New[string, bool](2000) // nolint: errcheck - this only returns an error if the size is less than 0

	return &eventEngineRepository{
		sharedRepository:    shared,
		m:                   m,
		createEventKeyCache: createEventKeyCache,
	}
}
