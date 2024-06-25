-- CreateTable
CREATE TABLE "WebhookWorker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" TEXT NOT NULL,
    "secret" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "tokenValue" TEXT,
    "deleted" BOOLEAN NOT NULL DEFAULT false,
    "tokenId" UUID,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "WebhookWorker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WebhookWorkerWorkflow" (
    "id" UUID NOT NULL,
    "webhookWorkerId" UUID NOT NULL,
    "workflowId" UUID NOT NULL,

    CONSTRAINT "WebhookWorkerWorkflow_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_id_key" ON "WebhookWorker"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_url_key" ON "WebhookWorker"("url");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_id_key" ON "WebhookWorkerWorkflow"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_webhookWorkerId_workflowId_key" ON "WebhookWorkerWorkflow"("webhookWorkerId", "workflowId");

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow"("id") ON DELETE CASCADE ON UPDATE CASCADE;
