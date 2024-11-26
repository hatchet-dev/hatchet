/*
  Warnings:

  - Made the column `parentId` on table `WorkflowTriggerCronRef` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" ALTER COLUMN "parentId" SET NOT NULL;
