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
    l.tenant_id = @tenantId::uuid
    AND l.task_id = @taskId::bigint
    AND l.task_inserted_at = @taskInsertedAt::timestamptz
    AND (sqlc.narg('search')::text IS NULL OR l.message iLIKE concat('%', sqlc.narg('search')::text, '%'))
ORDER BY
    l.created_at ASC
LIMIT COALESCE(sqlc.narg('limit'), 1000)
OFFSET COALESCE(sqlc.narg('offset'), 0);
