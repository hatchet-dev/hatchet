-- atlas:txmode none

-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "queue" text NOT NULL DEFAULT 'default';

-- Create index "StepRun_queue_createdAt_idx" to table: "StepRun"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepRun_queue_createdAt_idx" ON "StepRun" ("queue", "createdAt");
