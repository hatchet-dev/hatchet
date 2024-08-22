-- Create enum type "WebhookWorkerRequestMethod"
CREATE TYPE "WebhookWorkerRequestMethod" AS ENUM ('GET', 'POST', 'PUT');
-- Create "WebhookWorkerRequests" table
CREATE TABLE "WebhookWorkerRequests" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "webhookWorkerId" uuid NOT NULL, "statusCode" integer NOT NULL, "method" "WebhookWorkerRequestMethod" NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WebhookWorkerRequests_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WebhookWorkerRequests_id_key" to table: "WebhookWorkerRequests"
CREATE UNIQUE INDEX "WebhookWorkerRequests_id_key" ON "WebhookWorkerRequests" ("id");
