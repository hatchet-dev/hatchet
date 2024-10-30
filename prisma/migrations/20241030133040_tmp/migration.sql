/*
  Warnings:

  - The primary key for the `WorkflowTriggerCronRef` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - You are about to drop the column `id` on the `WorkflowTriggerCronRef` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" DROP CONSTRAINT "WorkflowTriggerCronRef_pkey",
DROP COLUMN "id";
