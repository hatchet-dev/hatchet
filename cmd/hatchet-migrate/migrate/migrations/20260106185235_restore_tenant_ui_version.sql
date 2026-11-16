-- +goose Up
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- NOTE: Down migration intentionally left empty.
-- This migration reintroduces the uiVersion column as part of a revert
-- strategy for databases that may have already run v1_0_67, which removed it.
-- We do not drop the column on rollback to avoid data loss and ensure
-- the column remains available going forward.
-- +goose StatementEnd
