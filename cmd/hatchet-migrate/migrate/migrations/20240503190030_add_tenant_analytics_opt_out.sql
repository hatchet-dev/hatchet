-- +goose Up
-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "analyticsOptOut" BOOLEAN NOT NULL DEFAULT false;
