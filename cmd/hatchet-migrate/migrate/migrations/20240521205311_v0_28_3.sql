-- +goose Up
-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "lastListenerEstablished" TIMESTAMP(3);
