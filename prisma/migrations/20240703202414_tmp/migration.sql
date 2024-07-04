/*
  Warnings:

  - You are about to drop the column `desiredWorkerAffinity` on the `Step` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "Step" DROP COLUMN "desiredWorkerAffinity";

-- CreateTable
CREATE TABLE "StepDesiredWorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,
    "required" BOOLEAN NOT NULL,
    "comparator" "WorkerLabelComparator" NOT NULL,
    "weight" INTEGER NOT NULL,

    CONSTRAINT "StepDesiredWorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel"("stepId");

-- CreateIndex
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel"("stepId", "key");

-- AddForeignKey
ALTER TABLE "StepDesiredWorkerLabel" ADD CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;
