/*
  Warnings:

  - The primary key for the `SemaphoreQueueItem` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - Changed the type of `stepRunId` on the `SemaphoreQueueItem` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- DropIndex
DROP INDEX "SemaphoreQueueItem_stepRunId_key";

-- AlterTable
ALTER TABLE "SemaphoreQueueItem" DROP CONSTRAINT "SemaphoreQueueItem_pkey",
DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT NOT NULL,
ADD CONSTRAINT "SemaphoreQueueItem_pkey" PRIMARY KEY ("stepRunId");
