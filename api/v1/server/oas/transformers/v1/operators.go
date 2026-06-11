package transformers

import (
	"encoding/json"
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ToV1HTTPOperator transforms a stored operator into its API representation. The signing
// secret stored in the config is intentionally never included in the response.
func ToV1HTTPOperator(op *sqlcv1.V1Operator) (gen.V1HTTPOperator, error) {
	var config httpoperator.HTTPOperatorConfig

	if err := json.Unmarshal(op.Config, &config); err != nil {
		return gen.V1HTTPOperator{}, err
	}

	result := gen.V1HTTPOperator{
		Metadata: gen.APIResourceMeta{
			Id:        op.ID.String(),
			CreatedAt: op.CreatedAt.Time,
			UpdatedAt: op.UpdatedAt.Time,
		},
		Name:                  op.Name,
		TenantId:              op.TenantID,
		TriggerEndpoint:       config.TriggerEndpoint,
		HealthcheckEndpoint:   config.HealthcheckEndpoint,
		RequestTimeoutSeconds: int32(config.RequestTimeoutSeconds), // #nosec G115 -- bounded config value
	}

	return result, nil
}

func ToV1HTTPOperatorList(operators []*sqlcv1.V1Operator, total, limit, offset int64) (gen.V1HTTPOperatorList, error) {
	rows := make([]gen.V1HTTPOperator, len(operators))

	for i, op := range operators {
		row, err := ToV1HTTPOperator(op)

		if err != nil {
			return gen.V1HTTPOperatorList{}, err
		}

		rows[i] = row
	}

	var currentPage, nextPage, totalPages int64

	if limit > 0 {
		currentPage = offset / limit
		nextPage = currentPage + 1
		totalPages = int64(math.Ceil(float64(total) / float64(limit)))
	}

	return gen.V1HTTPOperatorList{
		Rows: &rows,
		Pagination: &gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &totalPages,
		},
	}, nil
}
