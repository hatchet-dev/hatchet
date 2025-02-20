/*
  Warnings:

  - The primary key for the `SemaphoreQueueItem` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - You are about to drop the column `id` on the `SemaphoreQueueItem` table. All the data in the column will be lost.
  - A unique constraint covering the columns `[stepRunId]` on the table `SemaphoreQueueItem` will be added. If there are existing duplicate values, this will fail.

*/
-- DropIndex
DROP INDEX "SemaphoreQueueItem_stepRunId_workerId_key";

-- DropIndex
DROP INDEX "StepRun_jobRunId_status_tenantId_idx";

-- DropIndex
DROP INDEX "StepRun_tenantId_status_timeoutAt_idx";

-- AlterTable
ALTER TABLE "SemaphoreQueueItem" DROP CONSTRAINT "SemaphoreQueueItem_pkey",
DROP COLUMN "id",
ADD CONSTRAINT "SemaphoreQueueItem_pkey" PRIMARY KEY ("stepRunId");

-- AlterTable
ALTER TABLE "StepRunResultArchive" ADD COLUMN     "retryCount" INTEGER NOT NULL DEFAULT 0;

-- CreateIndex
CREATE UNIQUE INDEX "SemaphoreQueueItem_stepRunId_key" ON "SemaphoreQueueItem"("stepRunId");
