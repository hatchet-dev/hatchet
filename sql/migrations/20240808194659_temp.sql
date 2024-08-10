-- Create "File" table
CREATE TABLE "File" ("id" uuid NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "deletedAt" timestamp(3) NULL, "data" text NOT NULL, "tenantId" uuid NOT NULL, "additionalMetadata" jsonb NULL, PRIMARY KEY ("id"), CONSTRAINT "File_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "File_createdAt_idx" to table: "File"
CREATE INDEX "File_createdAt_idx" ON "File" ("createdAt");
-- Create index "File_id_key" to table: "File"
CREATE UNIQUE INDEX "File_id_key" ON "File" ("id");
-- Create index "File_tenantId_createdAt_idx" to table: "File"
CREATE INDEX "File_tenantId_createdAt_idx" ON "File" ("tenantId", "createdAt");
-- Create index "File_tenantId_idx" to table: "File"
CREATE INDEX "File_tenantId_idx" ON "File" ("tenantId");
