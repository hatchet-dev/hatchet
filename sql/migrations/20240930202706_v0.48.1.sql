-- Modify "JobRun" table
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_jobId_fkey";
-- Modify "JobRunLookupData" table
ALTER TABLE "JobRunLookupData" DROP CONSTRAINT "JobRunLookupData_tenantId_fkey";
