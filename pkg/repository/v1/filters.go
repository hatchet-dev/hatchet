package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type FilterRepository interface {
	CreateFilter(ctx context.Context, params sqlcv1.CreateFilterParams) (*sqlcv1.V1Filter, error)
	ListFilters(ctx context.Context, params sqlcv1.ListFiltersParams) ([]sqlcv1.V1Filter, error)
}

type filterRepository struct {
	*sharedRepository
}

func newFilterRepository(shared *sharedRepository) FilterRepository {
	return &filterRepository{
		sharedRepository: shared,
	}
}

func (r *filterRepository) CreateFilter(ctx context.Context, params sqlcv1.CreateFilterParams) (*sqlcv1.V1Filter, error) {
	return r.queries.CreateFilter(ctx, r.pool, params)
}
