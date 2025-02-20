-- CreateEnum
CREATE TYPE "LimitResource" AS ENUM ('WORKFLOW_RUN', 'EVENT', 'WORKER', 'CRON', 'SCHEDULE');

-- CreateEnum
CREATE TYPE "TenantResourceLimitAlertType" AS ENUM ('Alarm', 'Exhausted');

-- AlterTable
ALTER TABLE "TenantAlertingSettings" ADD COLUMN     "enableTenantResourceLimitAlerts" BOOLEAN NOT NULL DEFAULT true;

-- CreateTable
CREATE TABLE "TenantResourceLimit" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resource" "LimitResource" NOT NULL,
    "tenantId" UUID NOT NULL,
    "limitValue" INTEGER NOT NULL,
    "alarmValue" INTEGER,
    "value" INTEGER NOT NULL DEFAULT 0,
    "window" TEXT,
    "lastRefill" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "customValueMeter" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "TenantResourceLimit_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantResourceLimitAlert" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resourceLimitId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,
    "resource" "LimitResource" NOT NULL,
    "alertType" "TenantResourceLimitAlertType" NOT NULL,
    "value" INTEGER NOT NULL,
    "limit" INTEGER NOT NULL,

    CONSTRAINT "TenantResourceLimitAlert_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_id_key" ON "TenantResourceLimit"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_tenantId_resource_key" ON "TenantResourceLimit"("tenantId", "resource");

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimitAlert_id_key" ON "TenantResourceLimitAlert"("id");

-- AddForeignKey
ALTER TABLE "TenantResourceLimit" ADD CONSTRAINT "TenantResourceLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_resourceLimitId_fkey" FOREIGN KEY ("resourceLimitId") REFERENCES "TenantResourceLimit"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
