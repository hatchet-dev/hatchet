-- +goose Up
-- +goose StatementBegin
UPDATE "Tenant"
SET
    "version" = 'V1'
WHERE "version" = 'V0';

ALTER TABLE "Tenant"
    ALTER COLUMN "version" SET DEFAULT 'V1';

ALTER TABLE "Tenant"
    ADD CONSTRAINT "Tenant_version_not_v0" CHECK ("version" != 'V0');

-- Step 3: Drop uiVersion column and its type
ALTER TABLE "Tenant"
    DROP COLUMN "uiVersion";

DROP TYPE "TenantMajorUIVersion";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Step 1: Recreate the TenantMajorUIVersion type
CREATE TYPE "TenantMajorUIVersion" AS ENUM (
    'V0',
    'V1'
);

-- Step 2: Add back the uiVersion column
ALTER TABLE "Tenant"
    ADD COLUMN "uiVersion" "TenantMajorUIVersion" NOT NULL DEFAULT 'V0';

-- Step 3: Remove the version constraint and restore default
ALTER TABLE "Tenant"
    DROP CONSTRAINT "Tenant_version_not_v0";

ALTER TABLE "Tenant"
    ALTER COLUMN "version" SET DEFAULT 'V0';

-- Note: Cannot restore original V0 values for tenants that were migrated
-- +goose StatementEnd
