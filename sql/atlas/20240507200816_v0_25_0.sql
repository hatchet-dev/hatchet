-- CreateTable
CREATE TABLE
    "TenantAlertingSettings" (
        "id" UUID NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "deletedAt" TIMESTAMP(3),
        "tenantId" UUID NOT NULL,
        "maxFrequency" TEXT NOT NULL DEFAULT '1h',
        "lastAlertedAt" TIMESTAMP(3),
        "tickerId" UUID,
        CONSTRAINT "TenantAlertingSettings_pkey" PRIMARY KEY ("id")
    );

-- CreateTable
CREATE TABLE
    "TenantAlertEmailGroup" (
        "id" UUID NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "deletedAt" TIMESTAMP(3),
        "tenantId" UUID NOT NULL,
        "emails" TEXT NOT NULL,
        CONSTRAINT "TenantAlertEmailGroup_pkey" PRIMARY KEY ("id")
    );

-- CreateTable
CREATE TABLE
    "SlackAppWebhook" (
        "id" UUID NOT NULL,
        "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
        "deletedAt" TIMESTAMP(3),
        "tenantId" UUID NOT NULL,
        "teamId" TEXT NOT NULL,
        "teamName" TEXT NOT NULL,
        "channelId" TEXT NOT NULL,
        "channelName" TEXT NOT NULL,
        "webhookURL" BYTEA NOT NULL,
        CONSTRAINT "SlackAppWebhook_pkey" PRIMARY KEY ("id")
    );

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertingSettings_id_key" ON "TenantAlertingSettings" ("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertingSettings_tenantId_key" ON "TenantAlertingSettings" ("tenantId");

-- CreateIndex
CREATE UNIQUE INDEX "TenantAlertEmailGroup_id_key" ON "TenantAlertEmailGroup" ("id");

-- CreateIndex
CREATE UNIQUE INDEX "SlackAppWebhook_id_key" ON "SlackAppWebhook" ("id");

-- CreateIndex
CREATE UNIQUE INDEX "SlackAppWebhook_tenantId_teamId_channelId_key" ON "SlackAppWebhook" ("tenantId", "teamId", "channelId");

-- AddForeignKey
ALTER TABLE "TenantAlertingSettings" ADD CONSTRAINT "TenantAlertingSettings_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantAlertingSettings" ADD CONSTRAINT "TenantAlertingSettings_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker" ("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantAlertEmailGroup" ADD CONSTRAINT "TenantAlertEmailGroup_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "SlackAppWebhook" ADD CONSTRAINT "SlackAppWebhook_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Insert "TenantAlertingSettings" for every existing tenant
INSERT INTO
    "TenantAlertingSettings" ("id", "tenantId")
SELECT
    gen_random_uuid (),
    "id"
FROM
    "Tenant";