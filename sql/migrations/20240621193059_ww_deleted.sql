-- Modify "WebhookWorker" table
ALTER TABLE "WebhookWorker" ADD COLUMN "deleted" boolean NOT NULL DEFAULT false;
