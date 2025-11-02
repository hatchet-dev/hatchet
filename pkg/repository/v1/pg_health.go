package v1

import (
	"context"
)

type PGHealthRepository interface {
	CheckBloat(ctx context.Context) (int64, error)
}

type pgHealthRepository struct {
	*sharedRepository
}

func newPGHealthRepository(shared *sharedRepository) *pgHealthRepository {
	return &pgHealthRepository{
		sharedRepository: shared,
	}
}

func (h *pgHealthRepository) CheckBloat(ctx context.Context) (int64, error) {
	return h.queries.CheckBloat(context.Background(), h.pool)
}
