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
        UNNEST($3::TEXT[]) AS scope
)

SELECT f.id, f.tenant_id, f.workflow_id, f.scope, f.expression, f.payload, f.inserted_at, f.updated_at
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.scope) = (i.tenant_id, i.workflow_id, i.scope)
LIMIT COALESCE($4::BIGINT, 20000)
OFFSET COALESCE($5::BIGINT, 0)`

type ListFiltersParams struct {
	Tenantids    []pgtype.UUID `json:"tenantids"`
	Workflowids  []pgtype.UUID `json:"workflowids"`
	Scopes       []*string     `json:"scopes"`
	FilterLimit  *int64        `json:"limit"`
	FilterOffset *int64        `json:"offset"`
}

func (q *Queries) ListFilters(ctx context.Context, db DBTX, arg ListFiltersParams) ([]*V1Filter, error) {
	rows, err := db.Query(ctx, listFilters,
		arg.Tenantids,
		arg.Workflowids,
		arg.Scopes,
		arg.FilterLimit,
		arg.FilterOffset,
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
			&i.Scope,
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
