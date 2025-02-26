-- +goose Up
/*
  Warnings:

  - The primary key for the `WorkflowRun` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - A unique constraint covering the columns `[workflowRunId]` on the table `GetGroupKeyRun` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[id]` on the table `WorkflowRun` will be added. If there are existing duplicate values, this will fail.
  - Changed the type of `workflowRunId` on the `GetGroupKeyRun` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `workflowRunId` on the `JobRun` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `id` on the `WorkflowRun` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `parentId` on the `WorkflowRunTriggeredBy` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- DropForeignKey
ALTER TABLE "GetGroupKeyRun" DROP CONSTRAINT "GetGroupKeyRun_tenantId_workflowRunId_fkey";

-- DropForeignKey
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_workflowRunId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_parentId_fkey";

-- DropIndex
DROP INDEX "GetGroupKeyRun_tenantId_workflowRunId_key";

-- DropIndex
DROP INDEX "WorkflowRun_tenantId_id_key";

-- DropIndex
DROP INDEX "WorkflowRunTriggeredBy_tenantId_parentId_key";

-- AlterTable
ALTER TABLE "GetGroupKeyRun" DROP COLUMN "workflowRunId",
ADD COLUMN     "workflowRunId" UUID NOT NULL;

-- AlterTable
ALTER TABLE "JobRun" DROP COLUMN "workflowRunId",
ADD COLUMN     "workflowRunId" UUID NOT NULL;

-- AlterTable
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_pkey",
ADD COLUMN     "displayName" TEXT,
DROP COLUMN "id",
ADD COLUMN     "id" UUID NOT NULL,
ADD CONSTRAINT "WorkflowRun_pkey" PRIMARY KEY ("id");

-- AlterTable
ALTER TABLE "WorkflowRunTriggeredBy" ADD COLUMN     "input" JSONB,
DROP COLUMN "parentId",
ADD COLUMN     "parentId" UUID NOT NULL;

-- CreateIndex
CREATE UNIQUE INDEX "GetGroupKeyRun_workflowRunId_key" ON "GetGroupKeyRun"("workflowRunId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_id_key" ON "WorkflowRun"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_parentId_key" ON "WorkflowRunTriggeredBy"("parentId");

-- AddForeignKey
ALTER TABLE "GetGroupKeyRun" ADD CONSTRAINT "GetGroupKeyRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
