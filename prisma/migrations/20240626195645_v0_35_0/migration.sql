-- CreateTable
CREATE TABLE "SecurityCheckIdent" (
    "id" UUID NOT NULL,

    CONSTRAINT "SecurityCheckIdent_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent"("id");
