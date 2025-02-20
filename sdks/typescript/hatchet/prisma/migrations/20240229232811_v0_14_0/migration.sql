-- CreateEnum
CREATE TYPE "LogLineLevel" AS ENUM ('DEBUG', 'INFO', 'WARN', 'ERROR');

-- CreateTable
CREATE TABLE
    "LogLine" (
        "id" BIGSERIAL NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "tenantId" UUID NOT NULL,
        "stepRunId" UUID,
        "message" TEXT NOT NULL,
        "level" "LogLineLevel" NOT NULL DEFAULT 'INFO',
        "metadata" JSONB,
        CONSTRAINT "LogLine_pkey" PRIMARY KEY ("id")
    );

-- AddForeignKey
ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun" ("id") ON DELETE SET NULL ON UPDATE CASCADE;