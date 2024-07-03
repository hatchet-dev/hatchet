/*
  Warnings:

  - You are about to drop the `WorkerAffinity` table. If the table is not empty, all the data it contains will be lost.

*/
-- CreateEnum
CREATE TYPE "WorkerLabelComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');

-- DropForeignKey
ALTER TABLE "WorkerAffinity" DROP CONSTRAINT "WorkerAffinity_workerId_fkey";

-- DropTable
DROP TABLE "WorkerAffinity";

-- DropEnum
DROP TYPE "AffinityComparator";

-- CreateTable
CREATE TABLE "WorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workerId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,

    CONSTRAINT "WorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "WorkerLabel_workerId_idx" ON "WorkerLabel"("workerId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerLabel_workerId_key_key" ON "WorkerLabel"("workerId", "key");

-- AddForeignKey
ALTER TABLE "WorkerLabel" ADD CONSTRAINT "WorkerLabel_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
