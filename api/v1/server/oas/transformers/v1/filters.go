package transformers

import (
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
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
		Payload:      payload,
		ResourceHint: filter.ResourceHint,
		TenantId:     filter.TenantID.String(),
		WorkflowId:   uuid.MustParse(filter.WorkflowID.String()),
	}
}

func ToV1FilterList(filters []*sqlcv1.V1Filter) gen.V1FilterList {
	rows := make([]gen.V1Filter, len(filters))

	for i, filter := range filters {
		rows[i] = ToV1Filter(filter)
	}

	return gen.V1FilterList{
		Rows: &rows,
	}
}
