package prisma

import (
	"sync"
)

// workerCountCache stores a map of tenant ids to tenantWorkerCountCache
type workerCountCache struct {
	// map of tenant id (string) to worker count (*tenantWorkerCountCache)
	cachedWorkerCounts sync.Map
}

// tenantWorkerCountCache stores a map of worker ids to counts
type tenantWorkerCountCache struct {
	// wCache is a map of worker ids to counts
	wCache map[string]int

	// unprocessedAssigns is a map of worker ids to a count of assignments.
	// as we process assigned workers, this count is decremented. when the count reaches 0, the worker is removed.
	unprocessedAssigns map[string]int

	mu sync.RWMutex
}

func (w *workerCountCache) storeUnprocessedAssigns(tenantId string, decs map[string]int) {
	tenantCache := w.load(tenantId)

	tenantCache.mu.Lock()
	defer tenantCache.mu.Unlock()

	for workerId, dec := range decs {
		tenantCache.unprocessedAssigns[workerId] += dec
	}
}

// store overwrites the worker counts for the given tenant id
func (w *workerCountCache) store(tenantId string, workerToCounts map[string]int, processedAssigns map[string]int) {
	tenantCache := w.load(tenantId)

	tenantCache.mu.Lock()
	defer tenantCache.mu.Unlock()

	for workerId, count := range workerToCounts {
		tenantCache.wCache[workerId] = count
	}

	for workerId, dec := range processedAssigns {
		tenantCache.unprocessedAssigns[workerId] -= dec

		if tenantCache.unprocessedAssigns[workerId] <= 0 {
			delete(tenantCache.unprocessedAssigns, workerId)
		}
	}

	// delete any stored workers which are not in the workerToCounts or processedAssigns
	for workerId := range tenantCache.wCache {
		if _, ok := workerToCounts[workerId]; !ok {
			if _, ok := processedAssigns[workerId]; !ok {
				delete(tenantCache.wCache, workerId)
			}
		}
	}
}

func (w *workerCountCache) get(tenantId string, workerIds []string) map[string]int {
	tenantCache := w.load(tenantId)

	tenantCache.mu.RLock()
	defer tenantCache.mu.RUnlock()

	workerToCounts := make(map[string]int)

	for _, workerId := range workerIds {
		if count, ok := tenantCache.wCache[workerId]; ok {
			workerToCounts[workerId] = count
		}

		// remove unprocessed assigns
		if dec, ok := tenantCache.unprocessedAssigns[workerId]; ok {
			workerToCounts[workerId] -= dec
		}
	}

	return workerToCounts
}

func (w *workerCountCache) load(tenantId string) *tenantWorkerCountCache {
	cache, ok := w.cachedWorkerCounts.Load(tenantId)

	if !ok {
		cache = &tenantWorkerCountCache{
			wCache:             make(map[string]int),
			unprocessedAssigns: make(map[string]int),
		}

		w.cachedWorkerCounts.Store(tenantId, cache)
	}

	return cache.(*tenantWorkerCountCache)
}
