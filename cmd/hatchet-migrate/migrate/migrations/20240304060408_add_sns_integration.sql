-- +goose Up
-- CreateTable
CREATE TABLE "SNSIntegration" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "topicArn" TEXT NOT NULL,

    CONSTRAINT "SNSIntegration_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "SNSIntegration_id_key" ON "SNSIntegration"("id");

-- CreateIndex
CREATE UNIQUE INDEX "SNSIntegration_tenantId_topicArn_key" ON "SNSIntegration"("tenantId", "topicArn");

-- AddForeignKey
ALTER TABLE "SNSIntegration" ADD CONSTRAINT "SNSIntegration_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
