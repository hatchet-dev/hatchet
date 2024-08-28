-- CreateEnum
CREATE TYPE "WebhookWorkerRequestMethod" AS ENUM ('GET', 'POST', 'PUT');

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
