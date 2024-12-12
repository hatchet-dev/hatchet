-- +goose Up
/*
  Warnings:

  - You are about to drop the column `status` on the `Worker` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "Worker" DROP COLUMN "status";

-- DropEnum
DROP TYPE "WorkerStatus";
