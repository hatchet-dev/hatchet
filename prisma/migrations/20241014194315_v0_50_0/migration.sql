-- CreateEnum
CREATE TYPE "LeaseKind" AS ENUM ('WORKER', 'QUEUE');

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "insertOrder" INTEGER;

-- CreateTable
CREATE TABLE "Lease" (
    "id" BIGSERIAL NOT NULL,
    "expiresAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "resourceId" TEXT NOT NULL,
    "kind" "LeaseKind" NOT NULL,

    CONSTRAINT "Lease_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "Lease_tenantId_kind_resourceId_key" ON "Lease"("tenantId", "kind", "resourceId");

-- CreateEnum
CREATE TYPE "WorkerSDKS" AS ENUM ('UNKNOWN', 'GO', 'PYTHON', 'TYPESCRIPT');

-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "language" "WorkerSDKS",
ADD COLUMN     "languageVersion" TEXT,
ADD COLUMN     "os" TEXT,
ADD COLUMN     "runtimeExtra" TEXT,
ADD COLUMN     "sdkVersion" TEXT;
