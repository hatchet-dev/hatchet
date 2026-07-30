package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

const (
	workflowNameCacheSize = 100000
	workflowNameCacheTTL  = time.Hour
)

// workflowNameCache resolves workflow ids to workflow names for metric labels. It is shared
// across all tenants and queues in the scheduling pool.
type workflowNameCache struct {
	repo  v1.SchedulerRepository
	l     *zerolog.Logger
	cache *expirable.LRU[uuid.UUID, string]
}

func newWorkflowNameCache(repo v1.SchedulerRepository, l *zerolog.Logger) *workflowNameCache {
	return &workflowNameCache{
		repo:  repo,
		l:     l,
		cache: expirable.NewLRU[uuid.UUID, string](workflowNameCacheSize, nil, workflowNameCacheTTL),
	}
}

// resolve returns the names for the given workflow ids, fetching uncached ids from the
// database in a single query. Ids which cannot be resolved are absent from the result.
func (c *workflowNameCache) resolve(ctx context.Context, workflowIds []uuid.UUID) map[uuid.UUID]string {
	names := make(map[uuid.UUID]string, len(workflowIds))
	misses := make([]uuid.UUID, 0)

	for _, id := range workflowIds {
		if name, ok := c.cache.Get(id); ok {
			names[id] = name
		} else {
			misses = append(misses, id)
		}
	}

	if len(misses) == 0 {
		return names
	}

	fetched, err := c.repo.ListWorkflowNamesByIds(ctx, misses)

	if err != nil {
		c.l.Warn().Err(err).Msg("could not list workflow names for metric labels")
		return names
	}

	for id, name := range fetched {
		c.cache.Add(id, name)
		names[id] = name
	}

	return names
}
