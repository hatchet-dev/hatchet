-- CreateTable
CREATE TABLE "UserOAuth" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID NOT NULL,
    "provider" TEXT NOT NULL,
    "providerUserId" TEXT NOT NULL,
    "accessToken" TEXT NOT NULL,
    "refreshToken" TEXT,
    "expiresAt" TIMESTAMP(3),

    CONSTRAINT "UserOAuth_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_id_key" ON "UserOAuth"("id");

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_userId_key" ON "UserOAuth"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "UserOAuth_userId_provider_key" ON "UserOAuth"("userId", "provider");

-- AddForeignKey
ALTER TABLE "UserOAuth" ADD CONSTRAINT "UserOAuth_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;
