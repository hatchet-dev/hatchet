/*
  Warnings:

  - You are about to drop the column `gitRepoBranch` on the `WorkflowRun` table. All the data in the column will be lost.
  - You are about to drop the `GithubAppInstallation` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `GithubAppOAuth` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `GithubPullRequest` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `GithubPullRequestComment` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `GithubWebhook` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `WorkflowDeploymentConfig` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `_GithubAppInstallationToGithubWebhook` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `_GithubAppOAuthToUser` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `_GithubPullRequestToWorkflowRun` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "GithubAppInstallation" DROP CONSTRAINT "GithubAppInstallation_githubAppOAuthId_fkey";

-- DropForeignKey
ALTER TABLE "GithubAppInstallation" DROP CONSTRAINT "GithubAppInstallation_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "GithubAppInstallation" DROP CONSTRAINT "GithubAppInstallation_tenantVcsProviderId_fkey";

-- DropForeignKey
ALTER TABLE "GithubPullRequest" DROP CONSTRAINT "GithubPullRequest_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "GithubPullRequestComment" DROP CONSTRAINT "GithubPullRequestComment_pullRequestID_fkey";

-- DropForeignKey
ALTER TABLE "GithubPullRequestComment" DROP CONSTRAINT "GithubPullRequestComment_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "GithubWebhook" DROP CONSTRAINT "GithubWebhook_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowDeploymentConfig" DROP CONSTRAINT "WorkflowDeploymentConfig_githubAppInstallationId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowDeploymentConfig" DROP CONSTRAINT "WorkflowDeploymentConfig_workflowId_fkey";

-- DropForeignKey
ALTER TABLE "_GithubAppInstallationToGithubWebhook" DROP CONSTRAINT "_GithubAppInstallationToGithubWebhook_A_fkey";

-- DropForeignKey
ALTER TABLE "_GithubAppInstallationToGithubWebhook" DROP CONSTRAINT "_GithubAppInstallationToGithubWebhook_B_fkey";

-- DropForeignKey
ALTER TABLE "_GithubAppOAuthToUser" DROP CONSTRAINT "_GithubAppOAuthToUser_A_fkey";

-- DropForeignKey
ALTER TABLE "_GithubAppOAuthToUser" DROP CONSTRAINT "_GithubAppOAuthToUser_B_fkey";

-- DropForeignKey
ALTER TABLE "_GithubPullRequestToWorkflowRun" DROP CONSTRAINT "_GithubPullRequestToWorkflowRun_A_fkey";

-- DropForeignKey
ALTER TABLE "_GithubPullRequestToWorkflowRun" DROP CONSTRAINT "_GithubPullRequestToWorkflowRun_B_fkey";

-- AlterTable
ALTER TABLE "WorkflowRun" DROP COLUMN "gitRepoBranch";

-- DropTable
DROP TABLE "GithubAppInstallation";

-- DropTable
DROP TABLE "GithubAppOAuth";

-- DropTable
DROP TABLE "GithubPullRequest";

-- DropTable
DROP TABLE "GithubPullRequestComment";

-- DropTable
DROP TABLE "GithubWebhook";

-- DropTable
DROP TABLE "WorkflowDeploymentConfig";

-- DropTable
DROP TABLE "_GithubAppInstallationToGithubWebhook";

-- DropTable
DROP TABLE "_GithubAppOAuthToUser";

-- DropTable
DROP TABLE "_GithubPullRequestToWorkflowRun";

-- CreateTable
CREATE TABLE "WebhookWorker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" TEXT NOT NULL,
    "secret" TEXT NOT NULL,
    "url" TEXT NOT NULL,
    "tokenValue" TEXT,
    "deleted" BOOLEAN NOT NULL DEFAULT false,
    "tokenId" UUID,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "WebhookWorker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WebhookWorkerWorkflow" (
    "id" UUID NOT NULL,
    "webhookWorkerId" UUID NOT NULL,
    "workflowId" UUID NOT NULL,

    CONSTRAINT "WebhookWorkerWorkflow_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_id_key" ON "WebhookWorker"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorker_url_key" ON "WebhookWorker"("url");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_id_key" ON "WebhookWorkerWorkflow"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WebhookWorkerWorkflow_webhookWorkerId_workflowId_key" ON "WebhookWorkerWorkflow"("webhookWorkerId", "workflowId");

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tokenId_fkey" FOREIGN KEY ("tokenId") REFERENCES "APIToken"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorker" ADD CONSTRAINT "WebhookWorker_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_webhookWorkerId_fkey" FOREIGN KEY ("webhookWorkerId") REFERENCES "WebhookWorker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WebhookWorkerWorkflow" ADD CONSTRAINT "WebhookWorkerWorkflow_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow"("id") ON DELETE CASCADE ON UPDATE CASCADE;
