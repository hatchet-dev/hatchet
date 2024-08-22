/*
  Warnings:

  - You are about to drop the `WebhookWorkerRequests` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "WebhookWorkerRequests" DROP CONSTRAINT "WebhookWorkerRequests_webhookWorkerId_fkey";

-- DropTable
DROP TABLE "WebhookWorkerRequests";

-- CreateTable
CREATE TABLE "WebhookWorkerRequest" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "webhookWorkerId" UUID NOT NULL,
    "method" "WebhookWorkerRequestMethod" NOT NULL,
    "statusCode" INTEGER NOT NULL,

    CONSTRAINT "WebhookWorkerRequest_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerRequest_id_key" ON "WebhookWorkerRequest"("id");

-- AddForeignKey
ALTER TABLE "WebhookWorkerRequest" ADD CONSTRAINT "WebhookWorkerRequest_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
