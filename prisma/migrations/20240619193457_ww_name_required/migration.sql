/*
  Warnings:

  - Made the column `name` on table `WebhookWorker` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "WebhookWorker" ALTER COLUMN "name" SET NOT NULL;
