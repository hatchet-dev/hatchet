package transformers

import (
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToV1Filter(filter *sqlcv1.V1Filter) gen.V1Filter {
	var payload map[string]interface{}

	if filter.Payload != nil {
		payload = jsonToMap(filter.Payload)
	}

	return gen.V1Filter{
		Expression: filter.Expression,
		Metadata: gen.APIResourceMeta{
			CreatedAt: filter.InsertedAt.Time,
			UpdatedAt: filter.UpdatedAt.Time,
			Id:        filter.ID.String(),
		},
		Payload:       payload,
		Scope:         filter.Scope,
		TenantId:      filter.TenantID.String(),
		WorkflowId:    filter.WorkflowID,
		IsDeclarative: &filter.IsDeclarative,
	}
}

func ToV1FilterList(filters []*sqlcv1.V1Filter, total, limit, offset int64) gen.V1FilterList {
	rows := make([]gen.V1Filter, len(filters))

	for i, filter := range filters {
		rows[i] = ToV1Filter(filter)
	}

	currentPage := offset / limit
	nextPage := currentPage + 1
	totalPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V1FilterList{
		Rows: &rows,
		Pagination: &gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &totalPages,
		},
	}
}
