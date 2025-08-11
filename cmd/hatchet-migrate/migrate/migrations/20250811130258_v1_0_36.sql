-- +goose Up
-- +goose StatementBegin

UPDATE v1_task
SET priority = 3
WHERE priority > 3;

UPDATE v1_queue_item
SET priority = 3
WHERE priority > 3 AND retry_count = 0;

UPDATE v1_workflow_concurrency_slot
SET priority = 3
WHERE priority > 3;

UPDATE v1_concurrency_slot
SET priority = 3
WHERE priority > 3 AND task_retry_count = 0;

ALTER TABLE v1_task
ADD CONSTRAINT v1_task_priority_user_limit
CHECK (priority >= 1 AND priority <= 3);

ALTER TABLE v1_workflow_concurrency_slot
ADD CONSTRAINT v1_workflow_concurrency_slot_priority_user_limit
CHECK (priority >= 1 AND priority <= 3);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE v1_task DROP CONSTRAINT IF EXISTS v1_task_priority_user_limit;
ALTER TABLE v1_workflow_concurrency_slot DROP CONSTRAINT IF EXISTS v1_workflow_concurrency_slot_priority_user_limit;

-- +goose StatementEnd
