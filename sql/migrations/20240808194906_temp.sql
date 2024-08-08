-- Modify "File" table
ALTER TABLE "File" DROP COLUMN "data", ADD COLUMN "fileName" text NOT NULL, ADD COLUMN "filePath" text NOT NULL;
