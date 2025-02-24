-- Create enum type "WorkerSDKS"
CREATE TYPE "WorkerSDKS" AS ENUM ('UNKNOWN', 'GO', 'PYTHON', 'TYPESCRIPT');
-- Modify "Worker" table
ALTER TABLE "Worker" ADD COLUMN "language" "WorkerSDKS" NULL, ADD COLUMN "languageVersion" text NULL, ADD COLUMN "os" text NULL, ADD COLUMN "runtimeExtra" text NULL, ADD COLUMN "sdkVersion" text NULL;
