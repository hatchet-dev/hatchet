-- CreateTable
CREATE TABLE "EventKey" (
    "key" TEXT NOT NULL,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "EventKey_pkey" PRIMARY KEY ("key")
);

-- CreateIndex
CREATE UNIQUE INDEX "EventKey_key_tenantId_key" ON "EventKey"("key", "tenantId");
