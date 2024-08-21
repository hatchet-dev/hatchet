/*
  Warnings:

  - A unique constraint covering the columns `[actionId,tenantId]` on the table `Action` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[webhookId]` on the table `Worker` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateEnum
CREATE TYPE "WorkerType" AS ENUM ('WEBHOOK', 'MANAGED', 'SELFHOSTED');

-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "type" "WorkerType" NOT NULL DEFAULT 'SELFHOSTED',
ADD COLUMN     "webhookId" UUID;

-- CreateIndex
CREATE UNIQUE INDEX "Worker_webhookId_key" ON "Worker"("webhookId");

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_webhookId_fkey" FOREIGN KEY ("webhookId") REFERENCES "WebhookWorker"("id") ON DELETE SET NULL ON UPDATE CASCADE;
