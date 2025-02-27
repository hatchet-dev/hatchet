package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const listMatchConditionsForEvent = `-- name: ListMatchConditionsForEvent :many
WITH input AS (
    SELECT
        event_key, event_resource_hint
    FROM
        (
            SELECT
                unnest($3::text[]) AS event_key,
                -- NOTE: nullable field
                unnest($4::text[]) AS event_resource_hint
        ) AS subquery
)
SELECT
    v1_match_id,
    id,
    registered_at,
    event_type,
    m.event_key,
    m.event_resource_hint,
    readable_data_key,
    expression
FROM
    v1_match_condition m
JOIN
    input i ON (m.tenant_id, m.event_type, m.event_key, m.is_satisfied, m.event_resource_hint) = 
        ($1::uuid, $2::v1_event_type, i.event_key, FALSE, i.event_resource_hint)
`

type ListMatchConditionsForEventParams struct {
	Tenantid           pgtype.UUID   `json:"tenantid"`
	Eventtype          V1EventType   `json:"eventtype"`
	Eventkeys          []string      `json:"eventkeys"`
	Eventresourcehints []pgtype.Text `json:"eventresourcehints"`
}

type ListMatchConditionsForEventRow struct {
	V1MatchID         int64              `json:"v1_match_id"`
	ID                int64              `json:"id"`
	RegisteredAt      pgtype.Timestamptz `json:"registered_at"`
	EventType         V1EventType        `json:"event_type"`
	EventKey          string             `json:"event_key"`
	EventResourceHint pgtype.Text        `json:"event_resource_hint"`
	ReadableDataKey   string             `json:"readable_data_key"`
	Expression        pgtype.Text        `json:"expression"`
}

func (q *Queries) ListMatchConditionsForEvent(ctx context.Context, db DBTX, arg ListMatchConditionsForEventParams) ([]*ListMatchConditionsForEventRow, error) {
	rows, err := db.Query(ctx, listMatchConditionsForEvent,
		arg.Tenantid,
		arg.Eventtype,
		arg.Eventkeys,
		arg.Eventresourcehints,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*ListMatchConditionsForEventRow
	for rows.Next() {
		var i ListMatchConditionsForEventRow
		if err := rows.Scan(
			&i.V1MatchID,
			&i.ID,
			&i.RegisteredAt,
			&i.EventType,
			&i.EventKey,
			&i.EventResourceHint,
			&i.ReadableDataKey,
			&i.Expression,
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
