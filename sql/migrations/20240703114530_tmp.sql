-- Modify "WorkerAffinity" table
ALTER TABLE "WorkerAffinity" DROP COLUMN "value", ADD COLUMN "intValue" integer NULL, ADD COLUMN "strValue" text NULL;
