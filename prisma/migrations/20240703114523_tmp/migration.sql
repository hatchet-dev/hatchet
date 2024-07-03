/*
  Warnings:

  - You are about to drop the column `value` on the `WorkerAffinity` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "WorkerAffinity" DROP COLUMN "value",
ADD COLUMN     "intValue" INTEGER,
ADD COLUMN     "strValue" TEXT;
