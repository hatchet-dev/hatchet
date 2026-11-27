-- +goose Up
-- +goose StatementBegin
SET LOCAL lock_timeout = '60s';
UPDATE "Workflow" SET "isPaused" = FALSE WHERE "isPaused" IS NULL;
ALTER TABLE "Workflow" ALTER COLUMN "isPaused" SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Workflow" ALTER COLUMN "isPaused" DROP NOT NULL;
-- +goose StatementEnd
