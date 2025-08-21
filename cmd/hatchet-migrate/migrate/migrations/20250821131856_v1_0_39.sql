-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Tenant" ADD COLUMN "onboardingData" JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Tenant" DROP COLUMN "onboardingData";
-- +goose StatementEnd

