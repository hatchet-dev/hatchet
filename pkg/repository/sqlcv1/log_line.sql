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
    AND (sqlc.narg('search')::TEXT IS NULL OR l.message ILIKE CONCAT('%', sqlc.narg('search')::TEXT, '%'))
    AND (sqlc.narg('since')::TIMESTAMPTZ IS NULL OR l.created_at > sqlc.narg('since')::TIMESTAMPTZ)
    AND (sqlc.narg('until')::TIMESTAMPTZ IS NULL OR l.created_at < sqlc.narg('until')::TIMESTAMPTZ)
    AND (sqlc.narg('levels')::v1_log_line_level[] IS NULL OR l.level = ANY(sqlc.narg('levels')::v1_log_line_level[]))
    AND (sqlc.narg('attempt')::INTEGER IS NULL OR l.retry_count = (sqlc.narg('attempt')::INTEGER - 1))
ORDER BY
    CASE WHEN @orderByDirection::TEXT = 'DESC' THEN l.created_at END DESC,
    CASE WHEN @orderByDirection::TEXT = 'ASC' THEN l.created_at END ASC
LIMIT COALESCE(sqlc.narg('limit')::BIGINT, 1000)
OFFSET COALESCE(sqlc.narg('offset')::BIGINT, 0);
