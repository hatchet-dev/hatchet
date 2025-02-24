-- +goose Up
-- +goose StatementBegin
-- Create enum type "TenantMajorEngineVersion"
CREATE TYPE "TenantMajorEngineVersion" AS ENUM ('V0', 'V1');
-- Modify "Tenant" table
ALTER TABLE "Tenant" ADD COLUMN "version" "TenantMajorEngineVersion" NOT NULL DEFAULT 'V1';

ALTER TYPE "LeaseKind" ADD VALUE 'CONCURRENCY_STRATEGY';
-- +goose StatementEnd
