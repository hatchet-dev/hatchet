package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type FilterRepository interface {
	CreateFilter(ctx context.Context, tenantId string, params CreateFilterOpts) (*sqlcv1.V1Filter, error)
	ListFilters(ctx context.Context, tenantId string, params ListFiltersOpts) ([]*sqlcv1.V1Filter, error)
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
	Workflowid    pgtype.UUID `json:"workflowid" validate:"required,uuid"`
	Scope         string      `json:"scope" validate:"required"`
	Expression    string      `json:"expression" validate:"required"`
	Payload       []byte      `json:"payload"`
	IsDeclarative bool        `json:"is_declarative"`
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
	WorkflowIds  []pgtype.UUID `json:"workflow_ids" validate:"required"`
	Scopes       []*string     `json:"scopes"`
	FilterLimit  *int64        `json:"limit" validate:"omitnil,min=1"`
	FilterOffset *int64        `json:"offset" validate:"omitnil,min=0"`
}

type UpdateFilterOpts struct {
	Scope      *string `json:"scope"`
	Expression *string `json:"expression"`
	Payload    []byte  `json:"payload"`
}

func (r *filterRepository) ListFilters(ctx context.Context, tenantId string, opts ListFiltersOpts) ([]*sqlcv1.V1Filter, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.ListFilters(ctx, r.pool, sqlcv1.ListFiltersParams{
		Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
		Workflowids:  opts.WorkflowIds,
		Scopes:       opts.Scopes,
		FilterLimit:  opts.FilterLimit,
		FilterOffset: opts.FilterOffset,
	})
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
