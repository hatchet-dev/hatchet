-- atlas:txmode none

-- Add value to enum type: "StepRunStatus"
ALTER TYPE "StepRunStatus" ADD VALUE 'CANCELLING';
-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'SENT_TO_WORKER';
-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "queue" text NOT NULL DEFAULT 'default', ADD COLUMN "queueOrder" bigint NOT NULL DEFAULT 0;
-- Create index "StepRun_status_tenantId_deletedAt_queueOrder_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_status_tenantId_deletedAt_queueOrder_idx" ON "StepRun" ("status", "tenantId", "deletedAt", "queueOrder");
-- Create "StepRunPtr" table
CREATE TABLE "StepRunPtr" ("maxAssignedBlockAddr" bigint NOT NULL DEFAULT 0, "tenantId" uuid NOT NULL, PRIMARY KEY ("tenantId"));
-- Create index "StepRunPtr_tenantId_key" to table: "StepRunPtr"
CREATE UNIQUE INDEX "StepRunPtr_tenantId_key" ON "StepRunPtr" ("tenantId");
-- Create "StepRunQueue" table
CREATE TABLE "StepRunQueue" ("id" bigserial NOT NULL, "queue" text NOT NULL, "blockAddr" bigint NOT NULL, "tenantId" uuid NOT NULL, PRIMARY KEY ("id"));
-- Create index "StepRunQueue_tenantId_queue_key" to table: "StepRunQueue"
CREATE UNIQUE INDEX "StepRunQueue_tenantId_queue_key" ON "StepRunQueue" ("tenantId", "queue");

INSERT INTO
    "StepRunPtr" ("tenantId")
SELECT 
    "id" AS "tenantId"
FROM 
    "Tenant";

-- set the queue name to the step's action id for each step run in a pending assignment state
UPDATE
    "StepRun"
SET
    "queue" = "actionId",
    -- set the queue order to max bigint value to place currently step runs at the back of the queue 
    "queueOrder" = 9223372036854775807
FROM
    "Step"
WHERE
    "StepRun"."stepId" = "Step"."id" AND
    "StepRun"."status" = 'PENDING_ASSIGNMENT';