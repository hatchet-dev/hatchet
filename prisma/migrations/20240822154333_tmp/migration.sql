/*
  Warnings:

  - Changed the type of `method` on the `WebhookWorkerRequests` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- CreateEnum
CREATE TYPE "WebhookWorkerRequestMethod" AS ENUM ('GET', 'POST', 'PUT');

-- AlterTable
ALTER TABLE "WebhookWorkerRequests" DROP COLUMN "method",
ADD COLUMN     "method" "WebhookWorkerRequestMethod" NOT NULL;
