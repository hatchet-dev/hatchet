-- +goose Up
-- +goose StatementBegin
-- enum for environment
CREATE TYPE "TenantEnvironment" AS ENUM ('local', 'development', 'production');

ALTER TABLE "Tenant"
    ADD COLUMN "onboardingData" JSONB,
    ADD COLUMN "environment" "TenantEnvironment";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Tenant"
    DROP COLUMN "onboardingData",
    DROP COLUMN "environment";
DROP TYPE "TenantEnvironment";
-- +goose StatementEnd
