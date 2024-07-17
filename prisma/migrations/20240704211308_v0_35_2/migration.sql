-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "dataRetentionPeriod" TEXT NOT NULL DEFAULT '720h';
