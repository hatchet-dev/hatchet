-- Modify "WebhookWorker" table
ALTER TABLE "WebhookWorker" ADD COLUMN "tokenId" uuid NULL, ADD COLUMN "tokenValue" text NULL, ADD CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken" ("id") ON UPDATE CASCADE ON DELETE CASCADE;
