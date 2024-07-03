-- AlterTable
ALTER TABLE "WorkerAffinity" ADD COLUMN     "required" BOOLEAN NOT NULL DEFAULT false;

-- CreateIndex
CREATE INDEX "WorkerAffinity_workerId_idx" ON "WorkerAffinity"("workerId");
