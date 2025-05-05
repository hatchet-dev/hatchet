package transformers

import (
	"strconv"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToEventList(events []*sqlcv1.ListEventsRow) gen.EventList {
	rows := make([]gen.Event, len(events))

	pagination := gen.PaginationResponse{}

	for i, row := range events {
		additionalMetadata := jsonToMap(row.EventAdditionalMetadata)

		rows[i] = gen.Event{
			AdditionalMetadata: &additionalMetadata,
			Key:                row.EventKey,
			Metadata: gen.APIResourceMeta{
				CreatedAt: row.EventGeneratedAt.Time,
				UpdatedAt: row.EventGeneratedAt.Time,
				Id:        strconv.FormatInt(row.EventID, 10),
			},
		}
	}

	return gen.EventList{
		Rows:       rows,
		Pagination: pagination,
	}
}
