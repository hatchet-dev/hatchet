-- DropForeignKey
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_jobId_fkey";

-- DropForeignKey
ALTER TABLE "JobRunLookupData" DROP CONSTRAINT "JobRunLookupData_tenantId_fkey";
