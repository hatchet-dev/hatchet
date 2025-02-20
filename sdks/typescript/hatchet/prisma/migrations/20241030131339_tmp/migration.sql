/*
  Warnings:

  - A unique constraint covering the columns `[parentId,cron,name]` on the table `WorkflowTriggerCronRef` will be added. If there are existing duplicate values, this will fail.
  - The required column `id` was added to the `WorkflowTriggerCronRef` table with a prisma-level default value. This is not possible if the table is not empty. Please add this column as optional, then populate it before making it required.

*/
-- DropForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey";

-- AlterTable
ALTER TABLE "WorkflowRunTriggeredBy" ADD COLUMN     "cronName" TEXT;

-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN     "id" UUID NOT NULL,
ADD COLUMN     "name" TEXT,
ADD CONSTRAINT "WorkflowTriggerCronRef_pkey" PRIMARY KEY ("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerCronRef_parentId_cron_name_key" ON "WorkflowTriggerCronRef"("parentId", "cron", "name");

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") REFERENCES "WorkflowTriggerCronRef"("parentId", "cron", "name") ON DELETE SET NULL ON UPDATE CASCADE;
