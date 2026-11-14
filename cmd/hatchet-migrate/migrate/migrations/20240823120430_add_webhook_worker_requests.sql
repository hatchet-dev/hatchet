-- +goose Up
-- Create enum type "WebhookWorkerRequestMethod"
CREATE TYPE "WebhookWorkerRequestMethod" AS ENUM ('GET', 'POST', 'PUT');
-- Create "WebhookWorkerRequest" table
CREATE TABLE "WebhookWorkerRequest" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "webhookWorkerId" uuid NOT NULL, "method" "WebhookWorkerRequestMethod" NOT NULL, "statusCode" integer NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WebhookWorkerRequest_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WebhookWorkerRequest_id_key" to table: "WebhookWorkerRequest"
CREATE UNIQUE INDEX "WebhookWorkerRequest_id_key" ON "WebhookWorkerRequest" ("id");
