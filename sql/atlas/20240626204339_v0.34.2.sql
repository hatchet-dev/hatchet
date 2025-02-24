-- Create "ControllerPartition" table
CREATE TABLE "ControllerPartition" ("id" text NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "lastHeartbeat" timestamp(3) NULL, PRIMARY KEY ("id"));
-- Create index "ControllerPartition_id_key" to table: "ControllerPartition"
CREATE UNIQUE INDEX "ControllerPartition_id_key" ON "ControllerPartition" ("id");
-- Create "SecurityCheckIdent" table
CREATE TABLE "SecurityCheckIdent" ("id" uuid NOT NULL, PRIMARY KEY ("id"));
-- Create index "SecurityCheckIdent_id_key" to table: "SecurityCheckIdent"
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent" ("id");
-- Create "TenantWorkerPartition" table
CREATE TABLE "TenantWorkerPartition" ("id" text NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "lastHeartbeat" timestamp(3) NULL, PRIMARY KEY ("id"));
-- Create index "TenantWorkerPartition_id_key" to table: "TenantWorkerPartition"
CREATE UNIQUE INDEX "TenantWorkerPartition_id_key" ON "TenantWorkerPartition" ("id");
-- Modify "Tenant" table
ALTER TABLE "Tenant" ADD COLUMN "controllerPartitionId" text NULL, ADD COLUMN "workerPartitionId" text NULL, ADD CONSTRAINT "Tenant_controllerPartitionId_fkey" FOREIGN KEY ("controllerPartitionId") REFERENCES "ControllerPartition" ("id") ON UPDATE SET NULL ON DELETE SET NULL, ADD CONSTRAINT "Tenant_workerPartitionId_fkey" FOREIGN KEY ("workerPartitionId") REFERENCES "TenantWorkerPartition" ("id") ON UPDATE SET NULL ON DELETE SET NULL;
-- Create index "Tenant_controllerPartitionId_idx" to table: "Tenant"
CREATE INDEX "Tenant_controllerPartitionId_idx" ON "Tenant" ("controllerPartitionId");
-- Create index "Tenant_workerPartitionId_idx" to table: "Tenant"
CREATE INDEX "Tenant_workerPartitionId_idx" ON "Tenant" ("workerPartitionId");

INSERT INTO "SecurityCheckIdent" ("id") VALUES (gen_random_uuid());
