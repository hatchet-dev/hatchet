-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Tenant" ADD COLUMN IF NOT EXISTS "canUpgradeV1" BOOLEAN NOT NULL DEFAULT true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Tenant" DROP COLUMN IF EXISTS "canUpgradeV1";
-- +goose StatementEnd
