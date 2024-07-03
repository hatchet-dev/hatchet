/*
  Warnings:

  - A unique constraint covering the columns `[workerId,key]` on the table `WorkerAffinity` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateIndex
CREATE UNIQUE INDEX "WorkerAffinity_workerId_key_key" ON "WorkerAffinity"("workerId", "key");
