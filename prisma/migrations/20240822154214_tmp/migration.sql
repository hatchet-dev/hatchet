-- CreateTable
CREATE TABLE "WebhookWorkerRequests" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "webhookWorkerId" UUID NOT NULL,
    "method" TEXT NOT NULL,
    "statusCode" INTEGER NOT NULL,

    CONSTRAINT "WebhookWorkerRequests_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerRequests_id_key" ON "WebhookWorkerRequests"("id");

-- AddForeignKey
ALTER TABLE "WebhookWorkerRequests" ADD CONSTRAINT "WebhookWorkerRequests_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
