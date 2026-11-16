-- +goose Up
-- +goose StatementBegin
ALTER TYPE "TenantMemberRole" ADD VALUE 'VIEWER';
-- +goose StatementEnd
