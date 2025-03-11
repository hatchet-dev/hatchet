-- +goose Up
-- Create enum type "StickyStrategy"
CREATE TYPE "StickyStrategy" AS ENUM ('SOFT', 'HARD');
-- Create enum type "WorkerLabelComparator"
CREATE TYPE "WorkerLabelComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');
-- Modify "WorkflowVersion" table
ALTER TABLE "WorkflowVersion" ADD COLUMN "sticky" "StickyStrategy" NULL;
-- Create "StepDesiredWorkerLabel" table
CREATE TABLE "StepDesiredWorkerLabel" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "stepId" uuid NOT NULL, "key" text NOT NULL, "strValue" text NULL, "intValue" integer NULL, "required" boolean NOT NULL, "comparator" "WorkerLabelComparator" NOT NULL, "weight" integer NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "StepDesiredWorkerLabel_stepId_idx" to table: "StepDesiredWorkerLabel"
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel" ("stepId");
-- Create index "StepDesiredWorkerLabel_stepId_key_key" to table: "StepDesiredWorkerLabel"
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel" ("stepId", "key");
-- Create "WorkerLabel" table
CREATE TABLE "WorkerLabel" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "workerId" uuid NOT NULL, "key" text NOT NULL, "strValue" text NULL, "intValue" integer NULL, PRIMARY KEY ("id"), CONSTRAINT "WorkerLabel_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkerLabel_workerId_idx" to table: "WorkerLabel"
CREATE INDEX "WorkerLabel_workerId_idx" ON "WorkerLabel" ("workerId");
-- Create index "WorkerLabel_workerId_key_key" to table: "WorkerLabel"
CREATE UNIQUE INDEX "WorkerLabel_workerId_key_key" ON "WorkerLabel" ("workerId", "key");
-- Create "WorkflowRunDedupe" table
CREATE TABLE "WorkflowRunDedupe" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "tenantId" uuid NOT NULL, "workflowId" uuid NOT NULL, "workflowRunId" uuid NOT NULL, "value" text NOT NULL, CONSTRAINT "WorkflowRunDedupe_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkflowRunDedupe_id_key" to table: "WorkflowRunDedupe"
CREATE UNIQUE INDEX "WorkflowRunDedupe_id_key" ON "WorkflowRunDedupe" ("id");
-- Create index "WorkflowRunDedupe_tenantId_value_idx" to table: "WorkflowRunDedupe"
CREATE INDEX "WorkflowRunDedupe_tenantId_value_idx" ON "WorkflowRunDedupe" ("tenantId", "value");
-- Create index "WorkflowRunDedupe_tenantId_workflowId_value_key" to table: "WorkflowRunDedupe"
CREATE UNIQUE INDEX "WorkflowRunDedupe_tenantId_workflowId_value_key" ON "WorkflowRunDedupe" ("tenantId", "workflowId", "value");
-- Create "WorkflowRunStickyState" table
CREATE TABLE "WorkflowRunStickyState" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "tenantId" uuid NOT NULL, "workflowRunId" uuid NOT NULL, "desiredWorkerId" uuid NULL, "strategy" "StickyStrategy" NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WorkflowRunStickyState_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkflowRunStickyState_workflowRunId_key" to table: "WorkflowRunStickyState"
CREATE UNIQUE INDEX "WorkflowRunStickyState_workflowRunId_key" ON "WorkflowRunStickyState" ("workflowRunId");
