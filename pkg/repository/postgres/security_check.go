package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type securityCheckRepository struct {
	*sharedRepository
}

func NewSecurityCheckRepository(shared *sharedRepository) repository.SecurityCheckRepository {
	return &securityCheckRepository{
		sharedRepository: shared,
	}
}

func (a *securityCheckRepository) GetIdent() (string, error) {
	id, err := a.queries.GetSecurityCheckIdent(context.Background(), a.pool)

	if err != nil {
		return "", err
	}

	return id.String(), nil
}
