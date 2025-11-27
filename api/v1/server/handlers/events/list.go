package events

import (
	"github.com/google/uuid"

	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *EventService) EventList(ctx echo.Context, request gen.EventListRequestObject) (gen.EventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	total := int64(0)
	limit := int64(50)
	offset := int64(0)
	rows := make([]gen.Event, 0)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		limitInt := int(limit)
		offsetInt := int(offset)

		listOpts := &repository.ListEventOpts{
			Limit:  &limitInt,
			Offset: &offsetInt,
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
			limitInt = int(*request.Params.Limit)
			listOpts.Limit = &limitInt
		}

		if request.Params.Offset != nil {
			offsetInt = int(*request.Params.Offset)
			listOpts.Offset = &offsetInt
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
				splitValue := strings.SplitN(fmt.Sprintf("%v", v), ":", 2)

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

		for _, event := range listRes.Rows {
			eventData, err := transformers.ToEventFromSQLC(event)

			if err != nil {
				return nil, err
			}

			rows = append(rows, *eventData)
		}
	case dbsqlc.TenantMajorEngineVersionV1:
		since := time.Now().Add(-time.Hour * 24)

		if request.Params.Limit != nil {
			limit = *request.Params.Limit
		}

		if request.Params.Offset != nil {
			offset = *request.Params.Offset
		}

		opts := sqlcv1.ListEventsParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Limit: pgtype.Int8{
				Int64: limit,
				Valid: true,
			},
			Offset: pgtype.Int8{
				Int64: offset,
				Valid: true,
			},
			Since: pgtype.Timestamptz{
				Time:  since,
				Valid: true,
			},
		}

		if request.Params.Keys != nil {
			opts.Keys = *request.Params.Keys
		}

		if request.Params.Workflows != nil {
			workflowIds := make([]uuid.UUID, len(*request.Params.Workflows))

			for i, workflowId := range *request.Params.Workflows {
				workflowIds[i] = sqlchelpers.UUIDFromStr(workflowId)
			}

			opts.WorkflowIds = workflowIds
		}

		if request.Params.Statuses != nil {
			statuses := make([]string, len(*request.Params.Statuses))
			for i, status := range *request.Params.Statuses {
				statuses[i] = string(sqlcv1.V1ReadableStatusOlap(status))
			}
			opts.Statuses = statuses
		}

		if request.Params.EventIds != nil {
			eventIds := make([]uuid.UUID, len(*request.Params.EventIds))
			for i, eventId := range *request.Params.EventIds {
				eventIds[i] = sqlchelpers.UUIDFromStr(eventId.String())
			}
			opts.EventIds = eventIds
		}

		if request.Params.AdditionalMetadata != nil {
			additionalMeta := make(map[string]interface{})

			for _, m := range *request.Params.AdditionalMetadata {
				split := strings.SplitN(m, ":", 2)

				if len(split) != 2 {
					return nil, fmt.Errorf("invalid additional metadata format: %s, expected key:value", m)
				}

				key := split[0]
				value := split[1]

				if key == "" || value == "" {
					return nil, fmt.Errorf("invalid additional metadata format: %s, key and value must not be empty", m)
				}

				additionalMeta[key] = value
			}

			jsonbytes, err := json.Marshal(additionalMeta)

			if err != nil {
				return nil, fmt.Errorf("failed to marshal additional metadata: %w", err)
			}

			opts.AdditionalMetadata = jsonbytes
		}

		events, maybeTotal, err := t.config.V1.OLAP().ListEvents(ctx.Request().Context(), opts)

		if err != nil {
			return nil, fmt.Errorf("failed to list events: %w", err)
		}

		total = int64(len(events))

		if maybeTotal != nil {
			total = *maybeTotal
		}

		for _, event := range events {
			eventData, err := transformers.ToEventFromSQLCV1(event)
			if err != nil {
				return nil, err
			}
			rows = append(rows, *eventData)
		}
	}

	// use the total rows and limit to calculate the total pages
	totalPages := int64(math.Ceil(float64(total) / float64(limit)))
	currPage := 1 + int64(math.Ceil(float64(offset)/float64(limit)))
	nextPage := currPage + 1

	if currPage == totalPages {
		nextPage = currPage
	}

	return gen.EventList200JSONResponse(
		gen.EventList{
			Rows: &rows,
			Pagination: &gen.PaginationResponse{
				NumPages:    &totalPages,
				NextPage:    &nextPage,
				CurrentPage: &currPage,
			},
		},
	), nil
}
