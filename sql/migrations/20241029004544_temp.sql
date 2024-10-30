-- Drop index "StepRun_status_tenantId_idx" from table: "StepRun"
DROP INDEX "StepRun_status_tenantId_idx";
-- Modify "StepRun" table
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_new_pkey", ALTER COLUMN "startedAt" TYPE timestamp(3), ALTER COLUMN "finishedAt" TYPE timestamp(3), ALTER COLUMN "timeoutAt" TYPE timestamp(3), ALTER COLUMN "cancelledAt" TYPE timestamp(3), ADD PRIMARY KEY ("status", "id"), ADD CONSTRAINT "StepRun_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE, ADD CONSTRAINT "StepRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker" ("id") ON UPDATE CASCADE ON DELETE SET NULL;
-- Create index "StepRun_jobRunId_status_tenantId_idx" to table: "StepRun"
CREATE INDEX "StepRun_jobRunId_status_tenantId_idx" ON "StepRun" ("jobRunId", "status", "tenantId") WHERE (status = 'PENDING'::"StepRunStatus");
-- Rename an index from "StepRun_id_unique_idx" to "StepRun_id_key"
ALTER INDEX "StepRun_id_unique_idx" RENAME TO "StepRun_id_key";
