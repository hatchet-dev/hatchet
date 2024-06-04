/*
  Warnings:

  - The values [STEP_RUN] on the enum `LimitResource` will be removed. If these variants are still used in the database, this will fail.

*/
-- AlterEnum
BEGIN;
CREATE TYPE "LimitResource_new" AS ENUM ('WORKFLOW_RUN', 'EVENT', 'WORKER');
ALTER TABLE "TenantResourceLimit" ALTER COLUMN "resource" TYPE "LimitResource_new" USING ("resource"::text::"LimitResource_new");
ALTER TYPE "LimitResource" RENAME TO "LimitResource_old";
ALTER TYPE "LimitResource_new" RENAME TO "LimitResource";
DROP TYPE "LimitResource_old";
COMMIT;
