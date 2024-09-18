/*
  Warnings:

  - The `stepRunId` column on the `LogLine` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `stepRunId` column on the `QueueItem` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The primary key for the `StepRun` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - You are about to drop the column `tickerId` on the `StepRun` table. All the data in the column will be lost.
  - The `id` column on the `StepRun` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `stepRunId` column on the `StreamEvent` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `parentStepRunId` column on the `WorkflowRun` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `parentStepRunId` column on the `WorkflowTriggerScheduledRef` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The required column `id_uuid` was added to the `StepRun` table with a prisma-level default value. This is not possible if the table is not empty. Please add this column as optional, then populate it before making it required.
  - Changed the type of `stepRunId` on the `StepRunEvent` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `stepRunId` on the `StepRunResultArchive` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `stepRunId` on the `TimeoutQueueItem` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `A` on the `_StepRunOrder` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `B` on the `_StepRunOrder` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- DropForeignKey
ALTER TABLE "LogLine" DROP CONSTRAINT "LogLine_stepRunId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_jobRunId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_stepId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_tickerId_fkey";

-- DropForeignKey
ALTER TABLE "StepRunEvent" DROP CONSTRAINT "StepRunEvent_stepRunId_fkey";

-- DropForeignKey
ALTER TABLE "StepRunResultArchive" DROP CONSTRAINT "StepRunResultArchive_stepRunId_fkey";

-- DropForeignKey
ALTER TABLE "StreamEvent" DROP CONSTRAINT "StreamEvent_stepRunId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_parentStepRunId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" DROP CONSTRAINT "WorkflowTriggerScheduledRef_parentStepRunId_fkey";

-- DropForeignKey
ALTER TABLE "_StepRunOrder" DROP CONSTRAINT "_StepRunOrder_A_fkey";

-- DropForeignKey
ALTER TABLE "_StepRunOrder" DROP CONSTRAINT "_StepRunOrder_B_fkey";

-- DropIndex
DROP INDEX "StepRun_id_key";

-- AlterTable
ALTER TABLE "LogLine" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT;

-- AlterTable
ALTER TABLE "QueueItem" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT;

-- AlterTable
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_pkey",
DROP COLUMN "tickerId",
ADD COLUMN     "id_uuid" UUID NOT NULL,
DROP COLUMN "id",
ADD COLUMN     "id" BIGSERIAL NOT NULL,
ADD CONSTRAINT "StepRun_pkey" PRIMARY KEY ("id");

-- AlterTable
ALTER TABLE "StepRunEvent" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT NOT NULL;

-- AlterTable
ALTER TABLE "StepRunResultArchive" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT NOT NULL;

-- AlterTable
ALTER TABLE "StreamEvent" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT;

-- AlterTable
ALTER TABLE "TimeoutQueueItem" DROP COLUMN "stepRunId",
ADD COLUMN     "stepRunId" BIGINT NOT NULL;

-- AlterTable
ALTER TABLE "WorkflowRun" DROP COLUMN "parentStepRunId",
ADD COLUMN     "parentStepRunId" BIGINT;

-- AlterTable
ALTER TABLE "WorkflowTriggerScheduledRef" DROP COLUMN "parentStepRunId",
ADD COLUMN     "parentStepRunId" BIGINT;

-- AlterTable
ALTER TABLE "_StepRunOrder" DROP COLUMN "A",
ADD COLUMN     "A" BIGINT NOT NULL,
DROP COLUMN "B",
ADD COLUMN     "B" BIGINT NOT NULL;

-- CreateIndex
CREATE INDEX "StepRun_id_tenantId_idx" ON "StepRun"("id", "tenantId");

-- CreateIndex
CREATE INDEX "StepRunEvent_stepRunId_idx" ON "StepRunEvent"("stepRunId");

-- CreateIndex
CREATE INDEX "StepRunResultArchive_stepRunId_idx" ON "StepRunResultArchive"("stepRunId");

-- CreateIndex
CREATE INDEX "StreamEvent_stepRunId_idx" ON "StreamEvent"("stepRunId");

-- CreateIndex
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem"("stepRunId", "retryCount");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_parentId_parentStepRunId_childKey_key" ON "WorkflowRun"("parentId", "parentStepRunId", "childKey");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_parentId_parentStepRunId_childK_key" ON "WorkflowTriggerScheduledRef"("parentId", "parentStepRunId", "childKey");

-- CreateIndex
CREATE UNIQUE INDEX "_StepRunOrder_AB_unique" ON "_StepRunOrder"("A", "B");

-- CreateIndex
CREATE INDEX "_StepRunOrder_B_index" ON "_StepRunOrder"("B");

-- AddForeignKey
ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
