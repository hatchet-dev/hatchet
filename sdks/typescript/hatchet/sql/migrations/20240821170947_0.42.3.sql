-- Create enum type "WorkerType"
CREATE TYPE "WorkerType" AS ENUM ('WEBHOOK', 'MANAGED', 'SELFHOSTED');
-- Modify "Worker" table
ALTER TABLE "Worker" ADD COLUMN "type" "WorkerType" NOT NULL DEFAULT 'SELFHOSTED', ADD COLUMN "webhookId" uuid NULL, ADD CONSTRAINT "Worker_webhookId_fkey" FOREIGN KEY ("webhookId") REFERENCES "WebhookWorker" ("id") ON UPDATE CASCADE ON DELETE SET NULL;
-- Create index "Worker_webhookId_key" to table: "Worker"
CREATE UNIQUE INDEX "Worker_webhookId_key" ON "Worker" ("webhookId");
