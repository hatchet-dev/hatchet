-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
UPDATE "Workflow" SET "isPaused" = FALSE WHERE "isPaused" IS NULL;
ALTER TABLE "Workflow" ALTER COLUMN "isPaused" SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task ALTER COLUMN is_durable SET DEFAULT FALSE;
ALTER TABLE v1_task
  ADD CONSTRAINT v1_task_is_durable_not_null CHECK (is_durable IS NOT NULL) NOT VALID;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_task VALIDATE CONSTRAINT v1_task_is_durable_not_null;
ALTER TABLE v1_task ALTER COLUMN is_durable SET NOT NULL;
ALTER TABLE v1_task DROP CONSTRAINT v1_task_is_durable_not_null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Workflow" ALTER COLUMN "isPaused" DROP NOT NULL;
ALTER TABLE v1_task ALTER COLUMN is_durable DROP NOT NULL;
ALTER TABLE v1_task ALTER COLUMN is_durable DROP DEFAULT;
-- +goose StatementEnd
