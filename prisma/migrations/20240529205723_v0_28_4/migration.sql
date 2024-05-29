/*
  Warnings:

  - You are about to drop the `WorkerSemaphore` table. If the table is not empty, all the data it contains will be lost.
  - Made the column `maxRuns` on table `Worker` required. This step will fail if there are existing NULL values in that column.

*/
-- DropForeignKey
ALTER TABLE "WorkerSemaphore" DROP CONSTRAINT "WorkerSemaphore_workerId_fkey";

-- AlterTable
ALTER TABLE "Worker" ALTER COLUMN "maxRuns" SET NOT NULL,
ALTER COLUMN "maxRuns" SET DEFAULT 100;

-- DropTable
DROP TABLE "WorkerSemaphore";

-- CreateTable
CREATE TABLE "WorkerSemaphoreSlot" (
    "id" UUID NOT NULL,
    "workerId" UUID NOT NULL,
    "stepRunId" UUID,

    CONSTRAINT "WorkerSemaphoreSlot_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreSlot_id_key" ON "WorkerSemaphoreSlot"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphoreSlot_stepRunId_key" ON "WorkerSemaphoreSlot"("stepRunId");

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreSlot" ADD CONSTRAINT "WorkerSemaphoreSlot_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreSlot" ADD CONSTRAINT "WorkerSemaphoreSlot_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
