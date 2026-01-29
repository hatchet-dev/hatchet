-- +goose Up
-- +goose StatementBegin
-- NOTE: the default here is just to backfill previous incorrect cols
TRUNCATE TABLE "TenantResourceLimitAlert";
ALTER TABLE "TenantResourceLimitAlert"
    ADD COLUMN IF NOT EXISTS "resource" "LimitResource" NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Intentionally left empty since this migration is only for adding the column for incorrect previous migration
-- +goose StatementEnd
