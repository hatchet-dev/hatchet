-- +goose Up
-- Modify "Step" table
ALTER TABLE "Step" ADD COLUMN "retryBackoffFactor" double precision NULL, ADD COLUMN "retryMaxBackoff" integer NULL;
-- Create "RetryQueueItem" table
CREATE TABLE "RetryQueueItem" ("id" bigserial NOT NULL, "retryAfter" timestamp(3) NOT NULL, "stepRunId" uuid NOT NULL, "tenantId" uuid NOT NULL, "isQueued" boolean NOT NULL, PRIMARY KEY ("id"));
-- Create index "RetryQueueItem_isQueued_tenantId_retryAfter_idx" to table: "RetryQueueItem"
CREATE INDEX "RetryQueueItem_isQueued_tenantId_retryAfter_idx" ON "RetryQueueItem" ("isQueued", "tenantId", "retryAfter");
-- Create enum type "WorkflowTriggerCronRefMethods"
CREATE TYPE "WorkflowTriggerCronRefMethods" AS ENUM ('DEFAULT', 'API');
-- Create enum type "WorkflowTriggerScheduledRefMethods"
CREATE TYPE "WorkflowTriggerScheduledRefMethods" AS ENUM ('DEFAULT', 'API');

-- Step 1: Add the new columns with "id" as nullable
ALTER TABLE "WorkflowTriggerCronRef" 
ADD COLUMN "name" text NULL, 
ADD COLUMN "id" uuid NULL, 
ADD COLUMN "method" "WorkflowTriggerCronRefMethods" NOT NULL DEFAULT 'DEFAULT', 
ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_cron_name_key" UNIQUE ("parentId", "cron", "name");

-- Step 2: Populate "id" column with UUIDs for existing rows
UPDATE "WorkflowTriggerCronRef" 
SET "id" = gen_random_uuid() 
WHERE "id" IS NULL;

-- Step 3: Alter "id" column to be NOT NULL
ALTER TABLE "WorkflowTriggerCronRef" 
ALTER COLUMN "id" SET NOT NULL;

UPDATE "WorkflowTriggerCronRef" SET "name" = '' WHERE "name" IS NULL;

-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "method" "WorkflowTriggerScheduledRefMethods" NOT NULL DEFAULT 'DEFAULT';

-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" 
DROP CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey",
ADD COLUMN "cronName" text NULL;

ALTER TABLE "WorkflowRunTriggeredBy" 
ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" 
FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") 
REFERENCES "WorkflowTriggerCronRef" ("parentId", "cron", "name") 
ON UPDATE CASCADE 
ON DELETE SET NULL
NOT VALID;

-- Drop index "WorkflowTriggerCronRef_parentId_cron_key" from table: "WorkflowTriggerCronRef"
DROP INDEX "WorkflowTriggerCronRef_parentId_cron_key";
