-- Create sequences for bigint IDs
CREATE SEQUENCE IF NOT EXISTS "StepRun_id_seq";

-- Add new bigint columns
ALTER TABLE "LogLine" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "QueueItem" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "StepRun" ADD COLUMN "id_bigint" bigint;
ALTER TABLE "StepRunEvent" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "StepRunResultArchive" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "StreamEvent" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "TimeoutQueueItem" ADD COLUMN "stepRunId_bigint" bigint;
ALTER TABLE "WorkflowRun" ADD COLUMN "parentStepRunId_bigint" bigint;
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "parentStepRunId_bigint" bigint;
ALTER TABLE "_StepRunOrder" ADD COLUMN "A_bigint" bigint, ADD COLUMN "B_bigint" bigint;
ALTER TABLE "SemaphoreQueueItem" ADD COLUMN "stepRunId_bigint" bigint;

-- Update the new columns with converted values
UPDATE "StepRun" SET "id_bigint" = nextval('"StepRun_id_seq"');
UPDATE "LogLine" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "LogLine"."stepRunId" = "StepRun"."id";
UPDATE "QueueItem" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "QueueItem"."stepRunId" = "StepRun"."id";
UPDATE "StepRunEvent" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "StepRunEvent"."stepRunId" = "StepRun"."id";
UPDATE "StepRunResultArchive" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "StepRunResultArchive"."stepRunId" = "StepRun"."id";
UPDATE "StreamEvent" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "StreamEvent"."stepRunId" = "StepRun"."id";
UPDATE "TimeoutQueueItem" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "TimeoutQueueItem"."stepRunId" = "StepRun"."id";
UPDATE "WorkflowRun" SET "parentStepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "WorkflowRun"."parentStepRunId" = "StepRun"."id";
UPDATE "WorkflowTriggerScheduledRef" SET "parentStepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "WorkflowTriggerScheduledRef"."parentStepRunId" = "StepRun"."id";
UPDATE "_StepRunOrder" SET "A_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "_StepRunOrder"."A" = "StepRun"."id";
UPDATE "_StepRunOrder" SET "B_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "_StepRunOrder"."B" = "StepRun"."id";
UPDATE "SemaphoreQueueItem" SET "stepRunId_bigint" = "StepRun"."id_bigint" FROM "StepRun" WHERE "SemaphoreQueueItem"."stepRunId" = "StepRun"."id";


-- Drop all foreign key constraints referencing StepRun.id
ALTER TABLE "LogLine" DROP CONSTRAINT IF EXISTS "LogLine_stepRunId_fkey";
ALTER TABLE "QueueItem" DROP CONSTRAINT IF EXISTS "QueueItem_stepRunId_fkey";
ALTER TABLE "StepRunEvent" DROP CONSTRAINT IF EXISTS "StepRunEvent_stepRunId_fkey";
ALTER TABLE "StepRunResultArchive" DROP CONSTRAINT IF EXISTS "StepRunResultArchive_stepRunId_fkey";
ALTER TABLE "StreamEvent" DROP CONSTRAINT IF EXISTS "StreamEvent_stepRunId_fkey";
ALTER TABLE "TimeoutQueueItem" DROP CONSTRAINT IF EXISTS "TimeoutQueueItem_stepRunId_fkey";
ALTER TABLE "WorkflowRun" DROP CONSTRAINT IF EXISTS "WorkflowRun_parentStepRunId_fkey";
ALTER TABLE "WorkflowTriggerScheduledRef" DROP CONSTRAINT IF EXISTS "WorkflowTriggerScheduledRef_parentStepRunId_fkey";
DROP INDEX IF EXISTS "SemaphoreQueueItem_stepRunId_key";


---
-- Step 1: Rename stepRunId to stepRunId_uuid
ALTER TABLE "SemaphoreQueueItem" 
RENAME COLUMN "stepRunId" TO "stepRunId_uuid";

-- Step 2: Drop the existing primary key constraint and unique index
ALTER TABLE "SemaphoreQueueItem" 
DROP CONSTRAINT "SemaphoreQueueItem_pkey";

DROP INDEX IF EXISTS "SemaphoreQueueItem_pkey";

ALTER TABLE "SemaphoreQueueItem" 
ALTER COLUMN "stepRunId_bigint" SET NOT NULL;

ALTER TABLE "SemaphoreQueueItem" 
ADD PRIMARY KEY ("stepRunId_bigint");

-- Step 4: Rename stepRunId_bigint to stepRunId
ALTER TABLE "SemaphoreQueueItem" 
RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

-- Optional: Create an index on the new primary key if needed
CREATE INDEX "SemaphoreQueueItem_stepRunId_idx" 
ON "SemaphoreQueueItem"("stepRunId" int8_ops);

ALTER TABLE "SemaphoreQueueItem"
DROP COLUMN "stepRunId_uuid";
---

-- Drop constraints and indexes on StepRun
ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_jobRunId_fkey";
ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_stepId_fkey";
ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_tenantId_fkey";
DROP INDEX IF EXISTS "StepRun_id_key";

-- Now we can drop the primary key constraint
-- ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_pkey";

-- Now we can drop the old columns and rename the new ones
ALTER TABLE "LogLine" DROP COLUMN "stepRunId";
ALTER TABLE "LogLine" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "QueueItem" DROP COLUMN "stepRunId";
ALTER TABLE "QueueItem" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "StepRunEvent" DROP COLUMN "stepRunId";
ALTER TABLE "StepRunEvent" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "StepRunResultArchive" DROP COLUMN "stepRunId";
ALTER TABLE "StepRunResultArchive" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "StreamEvent" DROP COLUMN "stepRunId";
ALTER TABLE "StreamEvent" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "TimeoutQueueItem" DROP COLUMN "stepRunId";
ALTER TABLE "TimeoutQueueItem" RENAME COLUMN "stepRunId_bigint" TO "stepRunId";

ALTER TABLE "WorkflowRun" DROP COLUMN "parentStepRunId";
ALTER TABLE "WorkflowRun" RENAME COLUMN "parentStepRunId_bigint" TO "parentStepRunId";

ALTER TABLE "WorkflowTriggerScheduledRef" DROP COLUMN "parentStepRunId";
ALTER TABLE "WorkflowTriggerScheduledRef" RENAME COLUMN "parentStepRunId_bigint" TO "parentStepRunId";

ALTER TABLE "_StepRunOrder" DROP COLUMN "A";
ALTER TABLE "_StepRunOrder" DROP COLUMN "B";
ALTER TABLE "_StepRunOrder" RENAME COLUMN "A_bigint" TO "A";
ALTER TABLE "_StepRunOrder" RENAME COLUMN "B_bigint" TO "B";

-- Set the new column as the primary key for StepRun and add default
ALTER TABLE "StepRun" RENAME COLUMN "id" TO "id_uuid";
ALTER TABLE "StepRun" RENAME COLUMN "id_bigint" TO "id";


ALTER TABLE "StepRun" DROP CONSTRAINT IF EXISTS "StepRun_pkey";
ALTER TABLE "StepRun" ALTER COLUMN "id_uuid" DROP NOT NULL;
ALTER TABLE "StepRun" ADD PRIMARY KEY ("id");
ALTER TABLE "StepRun" ALTER COLUMN "id" SET DEFAULT nextval('"StepRun_id_seq"');

-- Recreate indexes
CREATE INDEX "StepRunResultArchive_stepRunId_idx" ON "StepRunResultArchive" ("stepRunId");
CREATE INDEX "StreamEvent_stepRunId_idx" ON "StreamEvent" ("stepRunId");

-- Recreate foreign key constraints
-- ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "StepRunEvent" ADD CONSTRAINT "StepRunEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "StepRunResultArchive" ADD CONSTRAINT "StepRunResultArchive_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "StreamEvent" ADD CONSTRAINT "StreamEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "TimeoutQueueItem" ADD CONSTRAINT "TimeoutQueueItem_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id");
-- ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey" FOREIGN KEY ("parentStepRunId") REFERENCES "StepRun"("id");

-- Drop the tickerId column from StepRun
ALTER TABLE "StepRun" DROP COLUMN IF EXISTS "tickerId";
