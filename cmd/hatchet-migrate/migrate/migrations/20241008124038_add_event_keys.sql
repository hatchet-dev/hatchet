-- +goose Up
-- Create "EventKey" table
CREATE TABLE "EventKey" ("key" text NOT NULL, "tenantId" uuid NOT NULL, PRIMARY KEY ("key"));
-- Create index "EventKey_key_tenantId_key" to table: "EventKey"
CREATE UNIQUE INDEX "EventKey_key_tenantId_key" ON "EventKey" ("key", "tenantId");
