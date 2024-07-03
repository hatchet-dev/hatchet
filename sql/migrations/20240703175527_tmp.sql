-- Modify "Step" table
ALTER TABLE "Step" ADD COLUMN "desiredWorkerAffinity" jsonb NULL;
-- Modify "WorkerAffinity" table
ALTER TABLE "WorkerAffinity" ALTER COLUMN "comparator" DROP DEFAULT, ALTER COLUMN "weight" DROP DEFAULT, ALTER COLUMN "required" DROP DEFAULT;
