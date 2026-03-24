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
    AND (sqlc.narg('taskIds')::BIGINT[] IS NULL OR l.task_id = ANY(sqlc.narg('taskIds')::BIGINT[]))
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

-- name: GetLogLinePointMetrics :many
SELECT
    DATE_BIN(
        COALESCE(sqlc.narg('interval')::INTERVAL, '1 minute'),
        created_at,
        TIMESTAMPTZ '1970-01-01 00:00:00+00'
    )::TIMESTAMPTZ AS minute_bucket,
    COUNT(*) FILTER (WHERE level = 'DEBUG') AS debug_count,
    COUNT(*) FILTER (WHERE level = 'INFO')  AS info_count,
    COUNT(*) FILTER (WHERE level = 'WARN')  AS warn_count,
    COUNT(*) FILTER (WHERE level = 'ERROR') AS error_count
FROM v1_log_line
WHERE
    tenant_id = @tenantId::UUID
    AND created_at BETWEEN @createdAfter::TIMESTAMPTZ AND @createdBefore::TIMESTAMPTZ
    AND (sqlc.narg('search')::TEXT IS NULL OR message ILIKE CONCAT('%', sqlc.narg('search')::TEXT, '%'))
    AND (sqlc.narg('levels')::v1_log_line_level[] IS NULL OR level = ANY(sqlc.narg('levels')::v1_log_line_level[]))
    AND (sqlc.narg('taskIds')::BIGINT[] IS NULL OR task_id = ANY(sqlc.narg('taskIds')::BIGINT[]))
GROUP BY minute_bucket
ORDER BY minute_bucket;
