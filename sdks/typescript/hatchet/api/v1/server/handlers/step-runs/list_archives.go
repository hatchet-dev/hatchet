package stepruns

import (
	"math"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func (t *StepRunService) StepRunListArchives(ctx echo.Context, request gen.StepRunListArchivesRequestObject) (gen.StepRunListArchivesResponseObject, error) {
	stepRun := ctx.Get("step-run").(*repository.GetStepRunFull)

	limit := 1000
	offset := 0

	listOpts := &repository.ListStepRunArchivesOpts{
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

	listRes, err := t.config.APIRepository.StepRun().ListStepRunArchives(
		sqlchelpers.UUIDToStr(stepRun.TenantId),
		sqlchelpers.UUIDToStr(stepRun.ID),
		listOpts,
	)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.StepRunArchive, len(listRes.Rows))

	for i := range listRes.Rows {
		e := listRes.Rows[i]

		eventData := transformers.ToStepRunArchive(e)

		rows[i] = *eventData
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listRes.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.StepRunListArchives200JSONResponse(
		gen.StepRunArchiveList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
