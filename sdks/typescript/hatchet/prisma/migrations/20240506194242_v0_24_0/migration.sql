/*
  Warnings:

  - A unique constraint covering the columns `[onFailureJobId]` on the table `WorkflowVersion` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateEnum
CREATE TYPE "JobKind" AS ENUM ('DEFAULT', 'ON_FAILURE');

-- AlterTable
ALTER TABLE "Event" ADD COLUMN     "additionalMetadata" JSONB;

-- AlterTable
ALTER TABLE "Job" ADD COLUMN     "kind" "JobKind" NOT NULL DEFAULT 'DEFAULT';

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "additionalMetadata" JSONB;

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "onFailureJobId" UUID;

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowVersion_onFailureJobId_key" ON "WorkflowVersion"("onFailureJobId");

-- AddForeignKey
ALTER TABLE "WorkflowVersion" ADD CONSTRAINT "WorkflowVersion_onFailureJobId_fkey" FOREIGN KEY ("onFailureJobId") REFERENCES "Job"("id") ON DELETE SET NULL ON UPDATE CASCADE;
