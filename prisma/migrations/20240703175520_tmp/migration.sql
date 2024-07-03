-- AlterTable
ALTER TABLE "Step" ADD COLUMN     "desiredWorkerAffinity" JSONB;

-- AlterTable
ALTER TABLE "WorkerAffinity" ALTER COLUMN "comparator" DROP DEFAULT,
ALTER COLUMN "weight" DROP DEFAULT,
ALTER COLUMN "required" DROP DEFAULT;
