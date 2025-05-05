package events

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *EventService) EventList(ctx echo.Context, request gen.EventListRequestObject) (gen.EventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	limit := 50
	offset := 0

	listOpts := &repository.ListEventOpts{
		Limit:  &limit,
		Offset: &offset,
	}

	if request.Params.Search != nil {
		listOpts.Search = request.Params.Search
	}

	if request.Params.Workflows != nil {
		listOpts.Workflows = *request.Params.Workflows
	}

	if request.Params.Keys != nil {
		listOpts.Keys = *request.Params.Keys
	}

	if request.Params.OrderByField != nil {
		listOpts.OrderBy = repository.StringPtr(string(*request.Params.OrderByField))
	}

	if request.Params.OrderByDirection != nil {
		listOpts.OrderDirection = repository.StringPtr(strings.ToUpper(string(*request.Params.OrderByDirection)))
	}

	if request.Params.Limit != nil {
		limit = int(*request.Params.Limit)
		listOpts.Limit = &limit
	}

	if request.Params.Offset != nil {
		offset = int(*request.Params.Offset)
		listOpts.Offset = &offset
	}

	if request.Params.Statuses != nil {
		statuses := make([]dbsqlc.WorkflowRunStatus, len(*request.Params.Statuses))

		for i, status := range *request.Params.Statuses {
			statuses[i] = dbsqlc.WorkflowRunStatus(status)
		}

		listOpts.WorkflowRunStatus = statuses
	}

	if request.Params.AdditionalMetadata != nil {
		additionalMetadata := make(map[string]interface{}, len(*request.Params.AdditionalMetadata))

		for _, v := range *request.Params.AdditionalMetadata {
			splitValue := strings.Split(fmt.Sprintf("%v", v), ":")

			if len(splitValue) == 2 {
				additionalMetadata[splitValue[0]] = splitValue[1]
			} else {
				return gen.EventList400JSONResponse(apierrors.NewAPIErrors("Additional metadata filters must be in the format key:value.")), nil

			}
		}

		additionalMetadataBytes, err := json.Marshal(additionalMetadata)

		if err != nil {
			return nil, err
		}

		listOpts.AdditionalMetadata = additionalMetadataBytes
	}

	if request.Params.EventIds != nil {
		eventIds := make([]string, len(*request.Params.EventIds))

		for i, id := range *request.Params.EventIds {
			idCp := id
			eventIds[i] = idCp.String()
		}

		listOpts.Ids = eventIds
	}

	dbCtx, cancel := context.WithTimeout(ctx.Request().Context(), 30*time.Second)
	defer cancel()

	listRes, err := t.config.APIRepository.Event().ListEvents(dbCtx, tenantId, listOpts)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.Event, len(listRes.Rows))

	for i, event := range listRes.Rows {
		eventData, err := transformers.ToEventFromSQLC(event)
		if err != nil {
			return nil, err
		}
		rows[i] = *eventData
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(listRes.Count) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.EventList200JSONResponse(
		gen.EventList{
			Rows: rows,
			Pagination: gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
