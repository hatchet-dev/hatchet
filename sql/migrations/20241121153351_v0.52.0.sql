-- Modify "Step" table
ALTER TABLE "Step" ADD COLUMN "retryBackoffFactor" double precision NULL, ADD COLUMN "retryMaxBackoff" integer NULL;
-- Create "RetryQueueItem" table
CREATE TABLE "RetryQueueItem" ("id" bigserial NOT NULL, "retryAfter" timestamp(3) NOT NULL, "stepRunId" uuid NOT NULL, "tenantId" uuid NOT NULL, "isQueued" boolean NOT NULL, PRIMARY KEY ("id"));
-- Create index "RetryQueueItem_isQueued_tenantId_retryAfter_idx" to table: "RetryQueueItem"
CREATE INDEX "RetryQueueItem_isQueued_tenantId_retryAfter_idx" ON "RetryQueueItem" ("isQueued", "tenantId", "retryAfter");
