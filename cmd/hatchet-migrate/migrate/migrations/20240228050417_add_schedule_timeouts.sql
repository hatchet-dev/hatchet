-- +goose Up
-- AlterTable
ALTER TABLE "GetGroupKeyRun" ADD COLUMN     "scheduleTimeoutAt" TIMESTAMP(3);

-- AlterTable
ALTER TABLE "Step" ADD COLUMN     "scheduleTimeout" TEXT NOT NULL DEFAULT '5m';

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "scheduleTimeout" TEXT NOT NULL DEFAULT '5m';
