-- +goose Up
-- +goose StatementBegin

ALTER TABLE "TenantResourceLimit" DROP COLUMN IF EXISTS "customValueMeter";

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE "TenantResourceLimit" ADD COLUMN IF NOT EXISTS "customValueMeter" boolean NOT NULL DEFAULT false;

-- +goose StatementEnd
