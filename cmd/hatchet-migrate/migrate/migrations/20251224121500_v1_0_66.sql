-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'Tenant_color_hex_check'
    ) THEN
        ALTER TABLE "Tenant"
        ADD CONSTRAINT "Tenant_color_hex_check"
        CHECK ("color" ~ '^#[0-9A-Fa-f]{6}$');
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Tenant"
DROP CONSTRAINT IF EXISTS "Tenant_color_hex_check";
-- +goose StatementEnd


