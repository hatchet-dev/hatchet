-- +goose Up
-- +goose StatementBegin
CREATE INDEX "TenantResourceLimitAlert_tenantId_idx" ON "TenantResourceLimitAlert" ("tenantId");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX "TenantResourceLimitAlert_tenantId_idx";
-- +goose StatementEnd
