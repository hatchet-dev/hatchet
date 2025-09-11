-- +goose Up
-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN     "enabled" BOOLEAN NOT NULL DEFAULT true;
