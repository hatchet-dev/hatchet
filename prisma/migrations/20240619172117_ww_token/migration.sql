-- AlterTable
ALTER TABLE "WebhookWorker" ADD COLUMN     "tokenId" UUID,
ADD COLUMN     "tokenValue" TEXT;

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken"("id") ON DELETE CASCADE ON UPDATE CASCADE;
