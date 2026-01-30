package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type FilterRepository interface {
	CreateFilter(ctx context.Context, tenantId string, params CreateFilterOpts) (*sqlcv1.V1Filter, error)
	ListFilters(ctx context.Context, tenantId string, params ListFiltersOpts) ([]*sqlcv1.V1Filter, int64, error)
	DeleteFilter(ctx context.Context, tenantId, filterId string) (*sqlcv1.V1Filter, error)
	GetFilter(ctx context.Context, tenantId, filterId string) (*sqlcv1.V1Filter, error)
	UpdateFilter(ctx context.Context, tenantId string, filterId string, opts UpdateFilterOpts) (*sqlcv1.V1Filter, error)
}

type filterRepository struct {
	*sharedRepository
}

func newFilterRepository(shared *sharedRepository) FilterRepository {
	return &filterRepository{
		sharedRepository: shared,
	}
}

type CreateFilterOpts struct {
	Workflowid    uuid.UUID `json:"workflowid" validate:"required,uuid"`
	Scope         string    `json:"scope" validate:"required"`
	Expression    string    `json:"expression" validate:"required"`
	Payload       []byte    `json:"payload"`
	IsDeclarative bool      `json:"is_declarative"`
}

func (r *filterRepository) CreateFilter(ctx context.Context, tenantId string, opts CreateFilterOpts) (*sqlcv1.V1Filter, error) {
	return r.queries.CreateFilter(ctx, r.pool, sqlcv1.CreateFilterParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Workflowid: opts.Workflowid,
		Scope:      opts.Scope,
		Expression: opts.Expression,
		Payload:    opts.Payload,
	})
}

type ListFiltersOpts struct {
	WorkflowIds []uuid.UUID `json:"workflow_ids"`
	Scopes      []string    `json:"scopes"`
	Limit       int64       `json:"limit" validate:"omitnil,min=1"`
	Offset      int64       `json:"offset" validate:"omitnil,min=0"`
}

type UpdateFilterOpts struct {
	Scope      *string `json:"scope"`
	Expression *string `json:"expression"`
	Payload    []byte  `json:"payload"`
}

func (r *filterRepository) ListFilters(ctx context.Context, tenantId string, opts ListFiltersOpts) ([]*sqlcv1.V1Filter, int64, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	filters, err := r.queries.ListFilters(ctx, r.pool, sqlcv1.ListFiltersParams{
		Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
		WorkflowIds:  opts.WorkflowIds,
		Scopes:       opts.Scopes,
		Filterlimit:  opts.Limit,
		Filteroffset: opts.Offset,
	})

	if err != nil {
		return nil, 0, err
	}

	filterCount, err := r.queries.CountFilters(ctx, r.pool, sqlcv1.CountFiltersParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		WorkflowIds: opts.WorkflowIds,
		Scopes:      opts.Scopes,
	})

	if err != nil {
		return nil, 0, err
	}

	return filters, filterCount, nil
}

func (r *filterRepository) DeleteFilter(ctx context.Context, tenantId, filterId string) (*sqlcv1.V1Filter, error) {
	return r.queries.DeleteFilter(ctx, r.pool, sqlcv1.DeleteFilterParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(filterId),
	})
}

func (r *filterRepository) GetFilter(ctx context.Context, tenantId, filterId string) (*sqlcv1.V1Filter, error) {
	return r.queries.GetFilter(ctx, r.pool, sqlcv1.GetFilterParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(filterId),
	})
}

func (r *filterRepository) UpdateFilter(ctx context.Context, tenantId string, filterId string, opts UpdateFilterOpts) (*sqlcv1.V1Filter, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateFilterParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(filterId),
		Payload:  opts.Payload,
	}

	if opts.Scope != nil {
		params.Scope = pgtype.Text{
			String: *opts.Scope,
			Valid:  true,
		}
	}

	if opts.Expression != nil {
		params.Expression = pgtype.Text{
			String: *opts.Expression,
			Valid:  true,
		}
	}

	return r.queries.UpdateFilter(ctx, r.pool, params)
}
