/*
  Warnings:

  - You are about to drop the `WorkerSemaphore` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `WorkerSemaphoreCount` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `WorkerSemaphoreSlot` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "WorkerSemaphore" DROP CONSTRAINT "WorkerSemaphore_workerId_fkey";

-- DropForeignKey
ALTER TABLE "WorkerSemaphoreCount" DROP CONSTRAINT "WorkerSemaphoreCount_workerId_fkey";

-- DropForeignKey
ALTER TABLE "WorkerSemaphoreSlot" DROP CONSTRAINT "WorkerSemaphoreSlot_stepRunId_fkey";

-- DropForeignKey
ALTER TABLE "WorkerSemaphoreSlot" DROP CONSTRAINT "WorkerSemaphoreSlot_workerId_fkey";

-- DropTable
DROP TABLE "WorkerSemaphore";

-- DropTable
DROP TABLE "WorkerSemaphoreCount";

-- DropTable
DROP TABLE "WorkerSemaphoreSlot";
