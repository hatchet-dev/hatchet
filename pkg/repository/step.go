package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type StepRepository interface {
	ListStepExpressions(ctx context.Context, stepId string) ([]*dbsqlc.StepExpression, error)
	ListReadableIds(ctx context.Context, stepIds []pgtype.UUID) (map[pgtype.UUID]string, error)
}
