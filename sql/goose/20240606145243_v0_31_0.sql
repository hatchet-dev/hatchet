-- +goose Up
-- Create enum type "LimitResource"
CREATE TYPE "LimitResource" AS ENUM ('WORKFLOW_RUN', 'EVENT', 'WORKER', 'CRON', 'SCHEDULE');
-- Create enum type "TenantResourceLimitAlertType"
CREATE TYPE "TenantResourceLimitAlertType" AS ENUM ('Alarm', 'Exhausted');
-- Modify "TenantAlertingSettings" table
ALTER TABLE "TenantAlertingSettings" ADD COLUMN "enableTenantResourceLimitAlerts" boolean NOT NULL DEFAULT true;
-- Create "TenantResourceLimit" table
CREATE TABLE "TenantResourceLimit" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "resource" "LimitResource" NOT NULL, "tenantId" uuid NOT NULL, "limitValue" integer NOT NULL, "alarmValue" integer NULL, "value" integer NOT NULL DEFAULT 0, "window" text NULL, "lastRefill" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "customValueMeter" boolean NOT NULL DEFAULT false, PRIMARY KEY ("id"), CONSTRAINT "TenantResourceLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "TenantResourceLimit_id_key" to table: "TenantResourceLimit"
CREATE UNIQUE INDEX "TenantResourceLimit_id_key" ON "TenantResourceLimit" ("id");
-- Create index "TenantResourceLimit_tenantId_resource_key" to table: "TenantResourceLimit"
CREATE UNIQUE INDEX "TenantResourceLimit_tenantId_resource_key" ON "TenantResourceLimit" ("tenantId", "resource");
-- Create "TenantResourceLimitAlert" table
CREATE TABLE "TenantResourceLimitAlert" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "resourceLimitId" uuid NOT NULL, "tenantId" uuid NOT NULL, "resource" "LimitResource" NOT NULL, "alertType" "TenantResourceLimitAlertType" NOT NULL, "value" integer NOT NULL, "limit" integer NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "TenantResourceLimitAlert_resourceLimitId_fkey" FOREIGN KEY ("resourceLimitId") REFERENCES "TenantResourceLimit" ("id") ON UPDATE CASCADE ON DELETE CASCADE, CONSTRAINT "TenantResourceLimitAlert_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "TenantResourceLimitAlert_id_key" to table: "TenantResourceLimitAlert"
CREATE UNIQUE INDEX "TenantResourceLimitAlert_id_key" ON "TenantResourceLimitAlert" ("id");
