package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type SecurityCheckRepository interface {
	GetIdent() (string, error)
}

type securityCheckRepository struct {
	*sharedRepository
}

func newSecurityCheckRepository(shared *sharedRepository) SecurityCheckRepository {
	return &securityCheckRepository{
		sharedRepository: shared,
	}
}

func (a *securityCheckRepository) GetIdent() (string, error) {
	id, err := a.queries.GetSecurityCheckIdent(context.Background(), a.pool)

	if err != nil {
		return "", err
	}

	return sqlchelpers.UUIDToStr(id), nil
}
