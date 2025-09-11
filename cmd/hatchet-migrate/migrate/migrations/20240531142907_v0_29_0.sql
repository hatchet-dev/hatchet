-- +goose Up
-- AlterTable
ALTER TABLE "APIToken" ADD COLUMN     "nextAlertAt" TIMESTAMP(3);
