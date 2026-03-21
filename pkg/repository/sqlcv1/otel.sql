-- name: InsertOtelSpans :exec
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@traceIds::BYTEA[]) AS trace_id,
        UNNEST(@spanIds::BYTEA[]) AS span_id,
        UNNEST(@parentSpanIds::TEXT[]) AS parent_span_id,
        UNNEST(@spanNames::TEXT[]) AS span_name,
        UNNEST(CAST(@spanKinds::TEXT[] AS v1_otel_span_kind[])) AS span_kind,
        UNNEST(@serviceNames::TEXT[]) AS service_name,
        UNNEST(CAST(@statusCodes::TEXT[] AS v1_otel_status_code[])) AS status_code,
        UNNEST(@statusMessages::TEXT[]) AS status_message,
        UNNEST(@durationNss::BIGINT[]) AS duration_ns,
        UNNEST(@resourceAttributes::JSONB[]) AS resource_attributes,
        UNNEST(@spanAttributes::JSONB[]) AS span_attributes,
        UNNEST(@scopeNames::TEXT[]) AS scope_name,
        UNNEST(@scopeVersions::TEXT[]) AS scope_version,
        UNNEST(@taskRunExternalIds::UUID[]) AS task_run_external_id,
        UNNEST(@workflowRunExternalIds::UUID[]) AS workflow_run_external_id,
        UNNEST(@retryCounts::INT[]) AS retry_count,
        UNNEST(@startTimes::TIMESTAMPTZ[]) AS start_time
)

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
)
SELECT
    tenant_id,
    trace_id,
    span_id,
    NULLIF(parent_span_id, '') AS parent_span_id,
    span_name,
    span_kind,
    service_name,
    status_code,
    NULLIF(status_message, '') AS status_message,
    duration_ns,
    NULLIF(resource_attributes, '{}'::JSONB) AS resource_attributes,
    NULLIF(span_attributes, '{}'::JSONB) AS span_attributes,
    NULLIF(scope_name, '') AS scope_name,
    NULLIF(scope_version, '') AS scope_version,
    NULLIF(task_run_external_id::TEXT, '00000000-0000-0000-0000-000000000000')::UUID AS task_run_external_id,
    NULLIF(workflow_run_external_id::TEXT, '00000000-0000-0000-0000-000000000000')::UUID AS workflow_run_external_id,
    retry_count,
    start_time
FROM inputs
ON CONFLICT (tenant_id, trace_id, start_time, span_id) DO NOTHING
;

-- name: InsertOTelTraceLookup :exec
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@externalIds::UUID[]) AS external_id,
        UNNEST(@retryCounts::INT[]) AS retry_count,
        UNNEST(@traceIds::BYTEA[]) AS trace_id,
        UNNEST(@startTimes::TIMESTAMPTZ[]) AS start_time
)
INSERT INTO v1_otel_trace_lookup_table (
    tenant_id,
    external_id,
    retry_count,
    trace_id,
    start_time
)
SELECT
    tenant_id,
    external_id,
    retry_count,
    trace_id,
    start_time
FROM inputs
ON CONFLICT (tenant_id, external_id, retry_count, start_time) DO NOTHING
;

-- name: LookUpTraceId :one
WITH candidate_traces AS (
    SELECT *
    FROM v1_otel_trace_lookup_table
    WHERE
        tenant_id = @tenantId::UUID
        AND external_id = @externalId::UUID
)

SELECT trace_id
FROM candidate_traces
-- get the max retry count + use time as a stable-ish order
ORDER BY retry_count DESC, start_time DESC
LIMIT 1
;


-- name: ListSpansByTraceId :many
SELECT *
FROM v1_otel_trace
WHERE
    tenant_id = @tenantId::UUID
    AND trace_id = @traceId::BYTEA
ORDER BY start_time ASC
OFFSET COALESCE(@spanOffset::BIGINT, 0)
LIMIT COALESCE(@spanLimit::BIGINT, 1000)
;
