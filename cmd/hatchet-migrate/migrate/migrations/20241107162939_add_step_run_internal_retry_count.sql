-- +goose Up
-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "internalRetryCount" integer NOT NULL DEFAULT 0;
