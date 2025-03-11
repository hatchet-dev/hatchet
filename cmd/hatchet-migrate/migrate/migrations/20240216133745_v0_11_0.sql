-- +goose Up
-- CreateEnum
CREATE TYPE "VcsProvider" AS ENUM ('GITHUB');

-- AlterTable
ALTER TABLE "Step" ADD COLUMN     "retries" INTEGER NOT NULL DEFAULT 0;

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "callerFiles" JSONB,
ADD COLUMN     "gitRepoBranch" TEXT,
ADD COLUMN     "retryCount" INTEGER NOT NULL DEFAULT 0;

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "gitRepoBranch" TEXT;

-- CreateTable
CREATE TABLE "WorkflowDeploymentConfig" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "workflowId" UUID NOT NULL,
    "gitRepoName" TEXT NOT NULL,
    "gitRepoOwner" TEXT NOT NULL,
    "gitRepoBranch" TEXT NOT NULL,
    "githubAppInstallationId" UUID,

    CONSTRAINT "WorkflowDeploymentConfig_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepRunResultArchive" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "stepRunId" UUID NOT NULL,
    "order" BIGSERIAL NOT NULL,
    "input" JSONB,
    "output" JSONB,
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,

    CONSTRAINT "StepRunResultArchive_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantVcsProvider" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "vcsProvider" "VcsProvider" NOT NULL,
    "config" JSONB,

    CONSTRAINT "TenantVcsProvider_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GithubAppInstallation" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "githubAppOAuthId" UUID NOT NULL,
    "installationId" INTEGER NOT NULL,
    "accountName" TEXT NOT NULL,
    "accountId" INTEGER NOT NULL,
    "accountAvatarURL" TEXT,
    "installationSettingsURL" TEXT,
    "config" JSONB,
    "tenantId" UUID,
    "tenantVcsProviderId" UUID,

    CONSTRAINT "GithubAppInstallation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GithubAppOAuth" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "githubUserID" INTEGER NOT NULL,
    "accessToken" BYTEA NOT NULL,
    "refreshToken" BYTEA,
    "expiresAt" TIMESTAMP(3),

    CONSTRAINT "GithubAppOAuth_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GithubPullRequest" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "repositoryOwner" TEXT NOT NULL,
    "repositoryName" TEXT NOT NULL,
    "pullRequestID" INTEGER NOT NULL,
    "pullRequestTitle" TEXT NOT NULL,
    "pullRequestNumber" INTEGER NOT NULL,
    "pullRequestHeadBranch" TEXT NOT NULL,
    "pullRequestBaseBranch" TEXT NOT NULL,
    "pullRequestState" TEXT NOT NULL,

    CONSTRAINT "GithubPullRequest_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GithubPullRequestComment" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "pullRequestID" UUID NOT NULL,
    "moduleID" TEXT NOT NULL,
    "commentID" INTEGER NOT NULL,

    CONSTRAINT "GithubPullRequestComment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "GithubWebhook" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "repositoryOwner" TEXT NOT NULL,
    "repositoryName" TEXT NOT NULL,
    "signingSecret" BYTEA NOT NULL,

    CONSTRAINT "GithubWebhook_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "_GithubAppInstallationToGithubWebhook" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_GithubAppOAuthToUser" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_GithubPullRequestToWorkflowRun" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowDeploymentConfig_id_key" ON "WorkflowDeploymentConfig"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowDeploymentConfig_workflowId_key" ON "WorkflowDeploymentConfig"("workflowId");

-- CreateIndex
CREATE UNIQUE INDEX "StepRunResultArchive_id_key" ON "StepRunResultArchive"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantVcsProvider_id_key" ON "TenantVcsProvider"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantVcsProvider_tenantId_vcsProvider_key" ON "TenantVcsProvider"("tenantId", "vcsProvider");

-- CreateIndex
CREATE UNIQUE INDEX "GithubAppInstallation_id_key" ON "GithubAppInstallation"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GithubAppInstallation_installationId_accountId_key" ON "GithubAppInstallation"("installationId", "accountId");

-- CreateIndex
CREATE UNIQUE INDEX "GithubAppOAuth_id_key" ON "GithubAppOAuth"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GithubAppOAuth_githubUserID_key" ON "GithubAppOAuth"("githubUserID");

-- CreateIndex
CREATE UNIQUE INDEX "GithubPullRequest_id_key" ON "GithubPullRequest"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GithubPullRequest_tenantId_repositoryOwner_repositoryName_p_key" ON "GithubPullRequest"("tenantId", "repositoryOwner", "repositoryName", "pullRequestNumber");

-- CreateIndex
CREATE UNIQUE INDEX "GithubPullRequestComment_id_key" ON "GithubPullRequestComment"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GithubWebhook_id_key" ON "GithubWebhook"("id");

-- CreateIndex
CREATE UNIQUE INDEX "GithubWebhook_tenantId_repositoryOwner_repositoryName_key" ON "GithubWebhook"("tenantId", "repositoryOwner", "repositoryName");

-- CreateIndex
CREATE UNIQUE INDEX "_GithubAppInstallationToGithubWebhook_AB_unique" ON "_GithubAppInstallationToGithubWebhook"("A", "B");

-- CreateIndex
CREATE INDEX "_GithubAppInstallationToGithubWebhook_B_index" ON "_GithubAppInstallationToGithubWebhook"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_GithubAppOAuthToUser_AB_unique" ON "_GithubAppOAuthToUser"("A", "B");

-- CreateIndex
CREATE INDEX "_GithubAppOAuthToUser_B_index" ON "_GithubAppOAuthToUser"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_GithubPullRequestToWorkflowRun_AB_unique" ON "_GithubPullRequestToWorkflowRun"("A", "B");

-- CreateIndex
CREATE INDEX "_GithubPullRequestToWorkflowRun_B_index" ON "_GithubPullRequestToWorkflowRun"("B");

-- AddForeignKey
ALTER TABLE "WorkflowDeploymentConfig" ADD CONSTRAINT "WorkflowDeploymentConfig_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowDeploymentConfig" ADD CONSTRAINT "WorkflowDeploymentConfig_githubAppInstallationId_fkey" FOREIGN KEY ("githubAppInstallationId") REFERENCES "GithubAppInstallation"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRunResultArchive" ADD CONSTRAINT "StepRunResultArchive_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantVcsProvider" ADD CONSTRAINT "TenantVcsProvider_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubAppInstallation" ADD CONSTRAINT "GithubAppInstallation_githubAppOAuthId_fkey" FOREIGN KEY ("githubAppOAuthId") REFERENCES "GithubAppOAuth"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubAppInstallation" ADD CONSTRAINT "GithubAppInstallation_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubAppInstallation" ADD CONSTRAINT "GithubAppInstallation_tenantVcsProviderId_fkey" FOREIGN KEY ("tenantVcsProviderId") REFERENCES "TenantVcsProvider"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubPullRequest" ADD CONSTRAINT "GithubPullRequest_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubPullRequestComment" ADD CONSTRAINT "GithubPullRequestComment_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubPullRequestComment" ADD CONSTRAINT "GithubPullRequestComment_pullRequestID_fkey" FOREIGN KEY ("pullRequestID") REFERENCES "GithubPullRequest"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "GithubWebhook" ADD CONSTRAINT "GithubWebhook_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubAppInstallationToGithubWebhook" ADD CONSTRAINT "_GithubAppInstallationToGithubWebhook_A_fkey" FOREIGN KEY ("A") REFERENCES "GithubAppInstallation"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubAppInstallationToGithubWebhook" ADD CONSTRAINT "_GithubAppInstallationToGithubWebhook_B_fkey" FOREIGN KEY ("B") REFERENCES "GithubWebhook"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubAppOAuthToUser" ADD CONSTRAINT "_GithubAppOAuthToUser_A_fkey" FOREIGN KEY ("A") REFERENCES "GithubAppOAuth"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubAppOAuthToUser" ADD CONSTRAINT "_GithubAppOAuthToUser_B_fkey" FOREIGN KEY ("B") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubPullRequestToWorkflowRun" ADD CONSTRAINT "_GithubPullRequestToWorkflowRun_A_fkey" FOREIGN KEY ("A") REFERENCES "GithubPullRequest"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_GithubPullRequestToWorkflowRun" ADD CONSTRAINT "_GithubPullRequestToWorkflowRun_B_fkey" FOREIGN KEY ("B") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

INSERT INTO
    "Tenant" (
        "id",
        "createdAt",
        "updatedAt",
        "deletedAt",
        "name",
        "slug"
    )
VALUES
    (
        '8d420720-ef03-41dc-9c73-1c93f276db97',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        NULL,
        'internal',
        'internal'
    ) ON CONFLICT DO NOTHING;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_internal_name_or_slug()
RETURNS trigger AS $$
BEGIN
  IF NEW."name" = 'internal' OR NEW."slug" = 'internal' THEN
    RAISE EXCEPTION 'Values "internal" for "name" or "slug" are not allowed.';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER check_name_or_slug_before_insert_or_update
BEFORE INSERT OR UPDATE ON "Tenant"
FOR EACH ROW EXECUTE FUNCTION prevent_internal_name_or_slug();
