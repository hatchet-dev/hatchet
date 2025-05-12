package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const listFilters = `-- name: ListFilters :many
WITH inputs AS (
    SELECT
        UNNEST($1::UUID[]) AS tenant_id,
        UNNEST($2::UUID[]) AS workflow_id,
        UNNEST($3::UUID[]) AS workflow_version_id,
        UNNEST($4::TEXT[]) AS resource_hint
)

SELECT f.id, f.tenant_id, f.workflow_id, f.resource_hint, f.expression, f.payload, f.inserted_at, f.updated_at
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.resource_hint) = (i.tenant_id, i.workflow_id, i.resource_hint)
`

type ListFiltersParams struct {
	Tenantids          []pgtype.UUID `json:"tenantids"`
	Workflowids        []pgtype.UUID `json:"workflowids"`
	Workflowversionids []pgtype.UUID `json:"workflowversionids"`
	Resourcehints      []*string     `json:"resourcehints"`
}

func (q *Queries) ListFilters(ctx context.Context, db DBTX, arg ListFiltersParams) ([]*V1Filter, error) {
	rows, err := db.Query(ctx, listFilters,
		arg.Tenantids,
		arg.Workflowids,
		arg.Workflowversionids,
		arg.Resourcehints,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*V1Filter
	for rows.Next() {
		var i V1Filter
		if err := rows.Scan(
			&i.ID,
			&i.TenantID,
			&i.WorkflowID,
			&i.ResourceHint,
			&i.Expression,
			&i.Payload,
			&i.InsertedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
