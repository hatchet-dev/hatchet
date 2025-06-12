package eventsv1

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func (t *V1EventsService) V1EventList(ctx echo.Context, request gen.V1EventListRequestObject) (gen.V1EventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	limit := int64(50)
	offset := int64(0)
	since := time.Now().Add(-time.Hour * 24)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	if request.Params.Since != nil {
		since = *request.Params.Since
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
		keys := make([]string, len(*request.Params.Keys))
		opts.Keys = keys
	}

	if request.Params.Until != nil {
		opts.Until = pgtype.Timestamptz{
			Time:  *request.Params.Until,
			Valid: true,
		}
	}

	if request.Params.WorkflowIds != nil {
		workflowIds := make([]pgtype.UUID, len(*request.Params.WorkflowIds))

		for i, workflowId := range *request.Params.WorkflowIds {
			workflowIds[i] = sqlchelpers.UUIDFromStr(workflowId.String())
		}

		opts.WorkflowIds = workflowIds
	}

	if request.Params.WorkflowRunStatuses != nil {
		statuses := make([]string, len(*request.Params.WorkflowRunStatuses))
		for i, status := range *request.Params.WorkflowRunStatuses {
			statuses[i] = string(sqlcv1.V1ReadableStatusOlap(status))
		}
		opts.Statuses = statuses
	}

	if request.Params.EventIds != nil {
		eventIds := make([]pgtype.UUID, len(*request.Params.EventIds))
		for i, eventId := range *request.Params.EventIds {
			eventIds[i] = sqlchelpers.UUIDFromStr(eventId.String())
		}
		opts.EventIds = eventIds
	}

	if request.Params.AdditionalMetadata != nil {
		additionalMeta := make(map[string]interface{})

		for _, m := range *request.Params.AdditionalMetadata {
			split := strings.Split(m, ":")

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

	total := int64(len(events))

	if maybeTotal != nil {
		total = *maybeTotal
	}

	rows := transformers.ToV1EventList(events, limit, offset, total)

	return gen.V1EventList200JSONResponse(
		rows,
	), nil
}
