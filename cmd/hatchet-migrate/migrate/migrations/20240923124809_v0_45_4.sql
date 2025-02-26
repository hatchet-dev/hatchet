-- +goose Up
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ALTER COLUMN "duration" TYPE bigint;
