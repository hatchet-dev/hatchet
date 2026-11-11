-- +goose Up
-- CreateTable
CREATE TABLE "StepRateLimit" (
    "units" INTEGER NOT NULL,
    "stepId" UUID NOT NULL,
    "rateLimitKey" TEXT NOT NULL,
    "tenantId" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "RateLimit" (
    "tenantId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "limitValue" INTEGER NOT NULL,
    "value" INTEGER NOT NULL,
    "window" TEXT NOT NULL,
    "lastRefill" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- CreateTable
CREATE TABLE "StreamEvent" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "stepRunId" UUID,
    "message" BYTEA NOT NULL,
    "metadata" JSONB,

    CONSTRAINT "StreamEvent_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "StepRateLimit_stepId_rateLimitKey_key" ON "StepRateLimit"("stepId", "rateLimitKey");

-- CreateIndex
CREATE UNIQUE INDEX "RateLimit_tenantId_key_key" ON "RateLimit"("tenantId", "key");

-- AddForeignKey
ALTER TABLE "StepRateLimit" ADD CONSTRAINT "StepRateLimit_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRateLimit" ADD CONSTRAINT "StepRateLimit_tenantId_rateLimitKey_fkey" FOREIGN KEY ("tenantId", "rateLimitKey") REFERENCES "RateLimit"("tenantId", "key") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRateLimit" ADD CONSTRAINT "StepRateLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "RateLimit" ADD CONSTRAINT "RateLimit_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StreamEvent" ADD CONSTRAINT "StreamEvent_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StreamEvent" ADD CONSTRAINT "StreamEvent_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_refill_value(rate_limit "RateLimit")
RETURNS INTEGER AS $$
DECLARE
    refill_amount INTEGER;
BEGIN
    IF NOW() - rate_limit."lastRefill" >= rate_limit."window"::INTERVAL THEN
        refill_amount := rate_limit."limitValue";
    ELSE
        refill_amount := rate_limit."value";
    END IF;
    RETURN refill_amount;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
