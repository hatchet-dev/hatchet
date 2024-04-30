/*
  Warnings:

  - You are about to drop the column `metadata` on the `Event` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "Event" DROP COLUMN "metadata",
ADD COLUMN     "additionalMetadata" JSONB;

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "additionalMetadata" JSONB;
