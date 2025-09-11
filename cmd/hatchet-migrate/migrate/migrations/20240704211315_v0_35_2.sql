-- +goose Up
-- Modify "Tenant" table
ALTER TABLE "Tenant" ADD COLUMN "dataRetentionPeriod" text NOT NULL DEFAULT '720h';
