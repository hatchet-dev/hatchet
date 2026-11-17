package operation

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// NewSharedGauge wraps a gauge so that its result for a resource is reused for one gauge tick.
// Pools that are given the same shared gauge then make one underlying call per resource per tick
// between them, instead of one call each. A pool may therefore act on a result that is up to one
// tick old. Errors are not kept, so the next caller retries.
func NewSharedGauge(gauge IntervalGauge) IntervalGauge {
	// no size limit: entries are removed once they expire, so the cache only holds resources
	// that were polled recently
	results := expirable.NewLRU[string, int](0, nil, gaugeInterval)

	return func(ctx context.Context, resourceId string) (int, error) {
		if res, ok := results.Get(resourceId); ok {
			return res, nil
		}

		res, err := gauge(ctx, resourceId)

		if err != nil {
			return 0, err
		}

		results.Add(resourceId, res)

		return res, nil
	}
}
