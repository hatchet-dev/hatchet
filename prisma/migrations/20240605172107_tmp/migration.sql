-- CreateEnum
CREATE TYPE "TenantResourceLimitAlertType" AS ENUM ('Alarm', 'Exhausted');

-- CreateTable
CREATE TABLE "TenantResourceLimitAlert" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "resourceLimitId" UUID NOT NULL,
    "resource" "LimitResource" NOT NULL,
    "alertType" "TenantResourceLimitAlertType" NOT NULL,
    "value" INTEGER NOT NULL,
    "limit" INTEGER NOT NULL,

    CONSTRAINT "TenantResourceLimitAlert_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "TenantResourceLimitAlert_id_key" ON "TenantResourceLimitAlert"("id");

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_resourceLimitId_fkey" FOREIGN KEY ("resourceLimitId") REFERENCES "TenantResourceLimit"("id") ON DELETE CASCADE ON UPDATE CASCADE;
