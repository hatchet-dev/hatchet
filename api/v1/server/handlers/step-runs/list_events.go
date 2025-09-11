package stepruns

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *StepRunService) StepRunListEvents(ctx echo.Context, request gen.StepRunListEventsRequestObject) (gen.StepRunListEventsResponseObject, error) {
	stepRun := ctx.Get("step-run").(*repository.GetStepRunFull)

	limit := 1000
	offset := 0

	listOpts := &repository.ListStepRunEventOpts{
		Limit:  &limit,
		Offset: &offset,
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		listOpts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		listOpts.Offset = &offset
	}

	listRes, err := t.config.APIRepository.StepRun().ListStepRunEvents(
		sqlchelpers.UUIDToStr(stepRun.ID),
		listOpts,
	)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.StepRunEvent, len(listRes.Rows))

	for i := range listRes.Rows {
		e := listRes.Rows[i]

		eventData := transformers.ToStepRunEvent(e)

		rows[i] = *eventData
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listRes.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.StepRunListEvents200JSONResponse(
		gen.StepRunEventList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
