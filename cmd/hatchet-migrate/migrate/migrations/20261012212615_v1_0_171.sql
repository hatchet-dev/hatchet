-- +goose Up
-- +goose StatementBegin
WITH replayed_tasks AS (
    SELECT DISTINCT durable_task_id, durable_task_inserted_at
    FROM v1_durable_event_log_branch_point
    WHERE durable_task_inserted_at >= '2026-08-20'
)

UPDATE v1_durable_event_log_entry e
SET triggered_at = e.inserted_at
FROM replayed_tasks t
WHERE (e.durable_task_id, e.durable_task_inserted_at) = (t.durable_task_id, t.durable_task_inserted_at)
  AND e.kind = 'RUN'
  AND e.branch_id > 1
  AND e.triggered_at IS NULL
  AND EXISTS (
      SELECT 1
      FROM v1_durable_event_log_entry prior
      WHERE (prior.durable_task_id, prior.durable_task_inserted_at) = (e.durable_task_id, e.durable_task_inserted_at)
        AND prior.child_task_external_id = e.child_task_external_id
        AND prior.triggered_at IS NOT NULL
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- intentionally blank
-- +goose StatementEnd
