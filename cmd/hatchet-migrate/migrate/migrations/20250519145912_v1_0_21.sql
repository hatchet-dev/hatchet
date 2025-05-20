-- +goose Up
-- +goose StatementBegin
CREATE TYPE "TenantMajorUIVersion" AS ENUM (
    'V0',
    'V1'
);

ALTER TABLE "Tenant"
ADD COLUMN "uiVersion" "TenantMajorUIVersion" NOT NULL DEFAULT 'V0'
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Tenant"
DROP COLUMN "uiVersion";

DROP TYPE "TenantMajorUIVersion"
-- +goose StatementEnd
