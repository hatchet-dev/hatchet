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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'TenantMajorUIVersion') THEN
        CREATE TYPE "TenantMajorUIVersion" AS ENUM ('V0', 'V1');
    END IF;
END
$$;

ALTER TABLE "Tenant"
    ADD COLUMN IF NOT EXISTS "uiVersion" "TenantMajorUIVersion" NOT NULL DEFAULT 'V0';

-- Step 3: Remove the version constraint and restore default
ALTER TABLE "Tenant"
    DROP CONSTRAINT "Tenant_version_not_v0";

ALTER TABLE "Tenant"
    ALTER COLUMN "version" SET DEFAULT 'V0';

-- Note: Cannot restore original V0 values for tenants that were migrated
-- +goose StatementEnd
