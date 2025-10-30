-- name: InsertLogLine :copyfrom
INSERT INTO v1_log_line (
    tenant_id,
    task_id,
    task_inserted_at,
    message,
    metadata,
    retry_count,
    level
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
);

-- name: ListLogLines :many
SELECT
    *
FROM
    v1_log_line l
WHERE
    l.tenant_id = @tenantId::UUID
    AND l.task_id = @taskId::BIGINT
    AND l.task_inserted_at = @taskInsertedAt::TIMESTAMPTZ
    AND (sqlc.narg('search')::TEXT IS NULL OR l.message iLIKE concat('%', sqlc.narg('search')::TEXT, '%'))
    AND (sqlc.narg('since')::TIMESTAMPTZ IS NULL OR l.created_at > sqlc.narg('since')::TIMESTAMPTZ)
ORDER BY
    l.created_at ASC
LIMIT COALESCE(sqlc.narg('limit'), 1000)
OFFSET COALESCE(sqlc.narg('offset'), 0);


-- name: CountLogLines :one
WITH filtered_logs AS (
    SELECT *
    FROM v1_log_line
    WHERE
        tenant_id = @tenantId::UUID
        AND task_id = @taskId::BIGINT
        AND task_inserted_at = @taskInsertedAt::TIMESTAMPTZ
    LIMIT 20000
)

SELECT COUNT(*)
FROM filtered_logs
;
