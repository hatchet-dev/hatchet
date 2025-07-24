-- name: ReadPayload :one
SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND type = @type::v1_payload_type
    AND task_id = @taskId::BIGINT
    AND task_inserted_at = @taskInsertedAt::TIMESTAMPTZ
;

-- name: ReadPayloads :many
WITH inputs AS (
    SELECT
        UNNEST(@taskIds::BIGINT[]) AS task_id,
        UNNEST(@taskInsertedAts::TIMESTAMPTZ[]) AS task_inserted_at,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type
)

SELECT *
FROM v1_payload
WHERE
    tenant_id = @tenantId::UUID
    AND (task_id, task_inserted_at, type) IN (
        SELECT task_id, task_inserted_at, type
        FROM inputs
    )
;

-- name: WritePayloads :exec
WITH inputs AS (
    SELECT
        UNNEST(@taskIds::BIGINT[]) AS task_id,
        UNNEST(@taskInsertedAts::TIMESTAMPTZ[]) AS task_inserted_at,
        UNNEST(CAST(@types::TEXT[] AS v1_payload_type[])) AS type,
        UNNEST(@payloads::JSONB[]) AS payload
)
INSERT INTO v1_payload (
    tenant_id,
    task_id,
    task_inserted_at,
    type,
    value
)
SELECT
    @tenantId::UUID,
    i.task_id,
    i.task_inserted_at,
    i.type,
    i.payload
FROM
    inputs i
;