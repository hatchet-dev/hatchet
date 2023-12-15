package repository

import (
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

type StepRepository interface {
	// ListStepsByActions returns a list of steps for a tenant which match the action ids.
	ListStepsByActions(tenantId string, actions []string) ([]db.StepModel, error)
}
