-- Create enum type "LeaseKind"
CREATE TYPE "LeaseKind" AS ENUM ('WORKER', 'QUEUE');
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ADD COLUMN "insertOrder" integer NULL;
-- Create "Lease" table
CREATE TABLE "Lease" ("id" bigserial NOT NULL, "expiresAt" timestamp(3) NULL, "tenantId" uuid NOT NULL, "resourceId" text NOT NULL, "kind" "LeaseKind" NOT NULL, PRIMARY KEY ("id"));
-- Create index "Lease_tenantId_kind_resourceId_key" to table: "Lease"
CREATE UNIQUE INDEX "Lease_tenantId_kind_resourceId_key" ON "Lease" ("tenantId", "kind", "resourceId");
-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'ACKNOWLEDGED';
