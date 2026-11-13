-- +goose Up
UPDATE "WorkflowTriggerCronRef" SET "name" = '' WHERE "name" IS NULL;
