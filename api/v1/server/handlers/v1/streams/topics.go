package streams

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// V1StreamTopicList lists the tenant's stream topics, or returns 403 when durable streams aren't enabled.
func (s *V1StreamsService) V1StreamTopicList(ctx echo.Context, request gen.V1StreamTopicListRequestObject) (gen.V1StreamTopicListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	reqCtx := ctx.Request().Context()

	entitled, err := s.config.V1.TenantEntitlement().HasEntitlement(reqCtx, tenant.ID, repository.EntitlementDurableStreams)

	if err != nil {
		return nil, err
	}

	if !entitled {
		return gen.V1StreamTopicList403JSONResponse(apierrors.NewAPIErrors(errNotEntitled.Error())), nil
	}

	limit := int64(50)
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if limit < 1 || limit > 1000 {
		return gen.V1StreamTopicList400JSONResponse(apierrors.NewAPIErrors("limit must be between 1 and 1000")), nil
	}

	if offset < 0 || offset > math.MaxInt32 {
		return gen.V1StreamTopicList400JSONResponse(apierrors.NewAPIErrors("offset is out of range")), nil
	}

	result, err := s.config.V1.Streams().ListTopics(reqCtx, tenant.ID, repository.ListStreamTopicsOpts{
		Search: request.Params.Search,
		Limit:  int32(limit),  // nolint: gosec
		Offset: int32(offset), // nolint: gosec
	})

	if err != nil {
		return nil, err
	}

	rows := make([]gen.V1StreamTopic, len(result.Rows))

	for i, row := range result.Rows {
		rows[i] = gen.V1StreamTopic{
			Namespace:       row.Namespace,
			Topic:           row.Topic,
			CreatedAt:       row.InsertedAt.Time,
			LastPublishedAt: row.LastPublishedAt.Time,
			MessageCount:    row.MessageCount,
		}
	}

	currentPage := offset/limit + 1
	numPages := (result.Total + limit - 1) / limit
	nextPage := min(currentPage+1, numPages)

	return gen.V1StreamTopicList200JSONResponse{
		Rows:           rows,
		RetentionHours: int(result.Retention.Hours()),
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}, nil
}
