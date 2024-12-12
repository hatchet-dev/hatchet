-- +goose Up
-- Modify "ControllerPartition" table
ALTER TABLE "ControllerPartition" ADD COLUMN "name" text NULL;
-- Modify "TenantWorkerPartition" table
ALTER TABLE "TenantWorkerPartition" ADD COLUMN "name" text NULL;
