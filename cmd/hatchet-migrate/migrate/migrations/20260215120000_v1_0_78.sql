-- +goose Up
-- Add RUBY to WorkerSDKS enum
ALTER TYPE "WorkerSDKS" ADD VALUE IF NOT EXISTS 'RUBY';

-- +goose Down
-- NOTE: Postgres does not support removing enum values.
-- A full enum recreation would be needed to revert this.
