package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type StepRepository interface {
	ListStepExpressions(ctx context.Context, stepId string) ([]*dbsqlc.StepExpression, error)
}
