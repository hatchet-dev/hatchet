/*
  Warnings:

  - Added the required column `tenantId` to the `TenantResourceLimitAlert` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "TenantResourceLimitAlert" ADD COLUMN     "tenantId" UUID NOT NULL;

-- AddForeignKey
ALTER TABLE "TenantResourceLimitAlert" ADD CONSTRAINT "TenantResourceLimitAlert_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
