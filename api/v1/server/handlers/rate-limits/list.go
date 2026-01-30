package rate_limits

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *RateLimitService) RateLimitList(ctx echo.Context, request gen.RateLimitListRequestObject) (gen.RateLimitListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID.String()

	limit := 50
	offset := 0

	listOpts := &v1.ListRateLimitOpts{
		Limit:  &limit,
		Offset: &offset,
	}

	if request.Params.Search != nil {
		listOpts.Search = request.Params.Search
	}

	if request.Params.OrderByField != nil {
		listOpts.OrderBy = v1.StringPtr(string(*request.Params.OrderByField))
	}

	if request.Params.OrderByDirection != nil {
		listOpts.OrderDirection = v1.StringPtr(strings.ToUpper(string(*request.Params.OrderByDirection)))
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		listOpts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		listOpts.Offset = &offset
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	listRes, err := t.config.V1.RateLimit().ListRateLimits(dbCtx, tenantId, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.RateLimit, len(listRes.Rows))

	for i, RateLimit := range listRes.Rows {
		RateLimitData, err := transformers.ToRateLimitFromSQLC(RateLimit)
		if err != nil {
			return nil, err
		}
		rows[i] = *RateLimitData
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listRes.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.RateLimitList200JSONResponse(
		gen.RateLimitList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
