package concurrencyv1

import (
	"errors"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ConcurrencyService) V1ConcurrencyKeyList(ctx echo.Context, request gen.V1ConcurrencyKeyListRequestObject) (gen.V1ConcurrencyKeyListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	var scope repository.ConcurrencyScope

	switch request.Scope {
	case gen.V1ConcurrencyScopeShared:
		scope = repository.ConcurrencyScopeShared
	case gen.V1ConcurrencyScopeWorkflow:
		scope = repository.ConcurrencyScopeWorkflow
	case gen.V1ConcurrencyScopeTask:
		scope = repository.ConcurrencyScopeTask
	default:
		return gen.V1ConcurrencyKeyList400JSONResponse(apierrors.NewAPIErrors("invalid scope")), nil
	}

	limit := int64(50)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if limit < 1 || limit > 1000 || offset < 0 {
		return gen.V1ConcurrencyKeyList400JSONResponse(apierrors.NewAPIErrors("limit must be between 1 and 1000 and offset must be non-negative")), nil
	}

	opts := repository.ListConcurrencyKeysOpts{
		Search: request.Params.Search,
		Limit:  limit,
		Offset: offset,
	}

	if request.Params.OnlyQueued != nil {
		opts.OnlyQueued = *request.Params.OnlyQueued
	}

	keys, total, err := t.config.V1.ConcurrencyOverview().ListKeys(ctx.Request().Context(), tenant.ID, scope, request.Strategy, opts)

	if errors.Is(err, repository.ErrConcurrencyStrategyNotFound) {
		return gen.V1ConcurrencyKeyList404JSONResponse(apierrors.NewAPIErrors("concurrency strategy not found")), nil
	}

	if err != nil {
		return nil, err
	}

	return gen.V1ConcurrencyKeyList200JSONResponse(
		transformers.ToV1ConcurrencyKeyList(keys, total, limit, offset),
	), nil
}
