-- Create "SecurityCheckIdent" table
CREATE TABLE "SecurityCheckIdent" ("id" uuid NOT NULL, PRIMARY KEY ("id"));
-- Create index "SecurityCheckIdent_id_key" to table: "SecurityCheckIdent"
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent" ("id");
-- Insert Default Ident
INSERT INTO "SecurityCheckIdent" ("id") VALUES (gen_random_uuid());