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
-- +goose StatementEnd
