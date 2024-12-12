-- +goose Up
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ADD COLUMN "duration" integer NULL;
