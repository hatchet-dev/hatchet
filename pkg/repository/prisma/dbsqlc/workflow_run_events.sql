
-- name: CreateWorkflowRunEvent :one

INSERT INTO "WorkflowRunEvent" (  "tenantId", "workflowRunId","eventType", "id")
VALUES ($1, $2, $3,  gen_random_uuid()) RETURNING *;

-- name: WorkflowRunEventsMetrics :many

SELECT
    minute, pending_count, queued_count, running_count, succeeded_count, retried_count, failed_count, queue_depth
FROM
    "WorkflowRunEventView"
WHERE
    "tenantId" = $1::uuid AND
    minute > $2::timestamp AND
    minute < $3::timestamp
ORDER BY minute;
