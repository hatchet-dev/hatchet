-- CreateEnum
CREATE TYPE "LimitResource" AS ENUM ('WORKFLOW_RUN', 'STEP_RUN', 'EVENT');

-- CreateTable
CREATE TABLE "TenantResourceLimit" (
    "id" UUID NOT NULL,
    "resource" "LimitResource" NOT NULL,
    "tenantId" UUID NOT NULL,
    "limitValue" INTEGER NOT NULL,
    "alarmValue" INTEGER,
    "value" INTEGER NOT NULL DEFAULT 0,
    "window" TEXT NOT NULL,
    "lastRefill" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "TenantResourceLimit_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_id_key" ON "TenantResourceLimit"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_tenantId_key" ON "TenantResourceLimit"("tenantId");

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimit_tenantId_resource_key" ON "TenantResourceLimit"("tenantId", "resource");

-- AddForeignKey
ALTER TABLE "TenantResourceLimit" ADD CONSTRAINT "TenantResourceLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
