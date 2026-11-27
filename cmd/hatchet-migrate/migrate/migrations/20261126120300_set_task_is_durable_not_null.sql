-- +goose Up
-- +goose StatementBegin
SET LOCAL lock_timeout = '60s';
ALTER TABLE v1_task ALTER COLUMN is_durable SET NOT NULL;
ALTER TABLE v1_task DROP CONSTRAINT v1_task_is_durable_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task ALTER COLUMN is_durable DROP NOT NULL;
ALTER TABLE v1_task
  ADD CONSTRAINT v1_task_is_durable_not_null CHECK (is_durable IS NOT NULL) NOT VALID;
-- +goose StatementEnd
