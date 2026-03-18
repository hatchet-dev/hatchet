-- name: InsertOtelSpans :copyfrom
INSERT INTO v1_otel_trace (
    tenant_id,
    trace_id,
    span_id,
    parent_span_id,
    span_name,
    span_kind,
    service_name,
    status_code,
    status_message,
    duration_ns,
    resource_attributes,
    span_attributes,
    scope_name,
    scope_version,
    task_run_external_id,
    workflow_run_external_id,
    retry_count,
    start_time
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
);

-- name: CountSpansByTaskExternalID :one
SELECT COUNT(*) FROM v1_otel_trace
WHERE tenant_id = @tenantId::UUID
AND task_run_external_id = @taskExternalId::UUID
AND retry_count = (
    SELECT MAX(retry_count) FROM v1_otel_trace
    WHERE tenant_id = @tenantId::UUID
    AND task_run_external_id = @taskExternalId::UUID
);

-- name: ListSpansByTaskExternalID :many
SELECT
    trace_id, span_id, parent_span_id, span_name, span_kind,
    service_name, status_code, status_message, duration_ns, start_time,
    resource_attributes, span_attributes, scope_name, scope_version,
    retry_count
FROM v1_otel_trace
WHERE tenant_id = @tenantId::UUID
AND task_run_external_id = @taskExternalId::UUID
AND retry_count = (
    SELECT MAX(retry_count) FROM v1_otel_trace
    WHERE tenant_id = @tenantId::UUID
    AND task_run_external_id = @taskExternalId::UUID
)
ORDER BY start_time ASC
OFFSET COALESCE(@spanOffset::BIGINT, 0)
LIMIT COALESCE(@spanLimit::BIGINT, 1000);

-- name: CountSpansByWorkflowRunExternalID :one
SELECT COUNT(*) FROM v1_otel_trace
WHERE tenant_id = @tenantId::UUID
AND workflow_run_external_id = @workflowRunExternalId::UUID
AND (
    task_run_external_id IS NULL
    OR (task_run_external_id, retry_count) IN (
        SELECT task_run_external_id, MAX(retry_count)
        FROM v1_otel_trace
        WHERE tenant_id = @tenantId::UUID
        AND workflow_run_external_id = @workflowRunExternalId::UUID
        AND task_run_external_id IS NOT NULL
        GROUP BY task_run_external_id
    )
);

-- name: ListSpansByWorkflowRunExternalID :many
SELECT
    trace_id, span_id, parent_span_id, span_name, span_kind,
    service_name, status_code, status_message, duration_ns, start_time,
    resource_attributes, span_attributes, scope_name, scope_version,
    retry_count
FROM v1_otel_trace
WHERE tenant_id = @tenantId::UUID
AND workflow_run_external_id = @workflowRunExternalId::UUID
AND (
    task_run_external_id IS NULL
    OR (task_run_external_id, retry_count) IN (
        SELECT task_run_external_id, MAX(retry_count)
        FROM v1_otel_trace
        WHERE tenant_id = @tenantId::UUID
        AND workflow_run_external_id = @workflowRunExternalId::UUID
        AND task_run_external_id IS NOT NULL
        GROUP BY task_run_external_id
    )
)
ORDER BY start_time ASC
OFFSET COALESCE(@spanOffset::BIGINT, 0)
LIMIT COALESCE(@spanLimit::BIGINT, 1000);

-- name: ListSpansByTraceIDs :many
SELECT
    trace_id, span_id, parent_span_id, span_name, span_kind,
    service_name, status_code, status_message, duration_ns, start_time,
    resource_attributes, span_attributes, scope_name, scope_version,
    retry_count
FROM v1_otel_trace
WHERE tenant_id = @tenantId::UUID
AND trace_id = ANY(@traceIds::TEXT[])
ORDER BY start_time ASC
LIMIT 10000;
