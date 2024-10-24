-- Create "SchedulerPartition" table
CREATE TABLE "SchedulerPartition" ("id" text NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "lastHeartbeat" timestamp(3) NULL, "name" text NULL, PRIMARY KEY ("id"));
-- Create index "SchedulerPartition_id_key" to table: "SchedulerPartition"
CREATE UNIQUE INDEX "SchedulerPartition_id_key" ON "SchedulerPartition" ("id");
-- Modify "Tenant" table
ALTER TABLE "Tenant" ADD COLUMN "schedulerPartitionId" text NULL, ADD CONSTRAINT "Tenant_schedulerPartitionId_fkey" FOREIGN KEY ("schedulerPartitionId") REFERENCES "SchedulerPartition" ("id") ON UPDATE SET NULL ON DELETE SET NULL;
