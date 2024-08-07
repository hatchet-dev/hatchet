-- Add value to enum type: "StepRunStatus"
ALTER TYPE "StepRunStatus" ADD VALUE 'CANCELLING';
-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'SENT_TO_WORKER';
-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "queue" text NOT NULL DEFAULT 'default', ADD COLUMN "queueOrder" bigint NOT NULL DEFAULT 0;
-- Create index "StepRun_status_tenantId_deletedAt_queueOrder_idx" to table: "StepRun"
CREATE INDEX "StepRun_status_tenantId_deletedAt_queueOrder_idx" ON "StepRun" ("status", "tenantId", "deletedAt", "queueOrder");
-- Create "Queue" table
CREATE TABLE "Queue" ("id" bigserial NOT NULL, "tenantId" uuid NOT NULL, "name" text NOT NULL, PRIMARY KEY ("id"));
-- Create index "Queue_tenantId_name_key" to table: "Queue"
CREATE UNIQUE INDEX "Queue_tenantId_name_key" ON "Queue" ("tenantId", "name");
-- Create "QueueItem" table
CREATE TABLE "QueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NULL, "stepId" uuid NULL, "actionId" text NULL, "scheduleTimeoutAt" timestamp(3) NULL, "stepTimeout" text NULL, "isQueued" boolean NOT NULL, "tenantId" uuid NOT NULL, "queue" text NOT NULL, PRIMARY KEY ("id"));
-- Create index "QueueItem_isQueued_queue_idx" to table: "QueueItem"
CREATE INDEX "QueueItem_isQueued_queue_id_idx" ON "QueueItem" ("isQueued", "queue", "id");
-- Create "StepRunPtr" table
CREATE TABLE "StepRunPtr" ("maxAssignedBlockAddr" bigint NOT NULL DEFAULT 0, "tenantId" uuid NOT NULL, PRIMARY KEY ("tenantId"));
-- Create index "StepRunPtr_tenantId_key" to table: "StepRunPtr"
CREATE UNIQUE INDEX "StepRunPtr_tenantId_key" ON "StepRunPtr" ("tenantId");
-- Create "StepRunQueue" table
CREATE TABLE "StepRunQueue" ("id" bigserial NOT NULL, "queue" text NOT NULL, "blockAddr" bigint NOT NULL, "tenantId" uuid NOT NULL, PRIMARY KEY ("id"));
-- Create index "StepRunQueue_tenantId_queue_key" to table: "StepRunQueue"
CREATE UNIQUE INDEX "StepRunQueue_tenantId_queue_key" ON "StepRunQueue" ("tenantId", "queue");