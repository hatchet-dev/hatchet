-- CreateTable
CREATE TABLE "SecurityCheckIdent" (
    "id" UUID NOT NULL DEFAULT uuid_generate_v4(),

    CONSTRAINT "SecurityCheckIdent_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent"("id");

-- Insert Default Ident
INSERT INTO "SecurityCheckIdent" ("id") VALUES (gen_random_uuid());