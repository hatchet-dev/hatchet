import { Separator } from '@/next/components/ui/separator';
import BasicLayout from '@/next/components/layouts/basic.layout';
import {
  Headline,
  PageTitle,
  HeadlineActions,
  HeadlineActionItem,
} from '@/next/components/ui/page-header';
import { Button } from '@/next/components/ui/button';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/next/components/ui/dialog';
import { DataTable } from '@/next/components/ui/data-table';
import { columns as githubInstallationsColumns } from './components/github-installations-columns';
import {
  GithubIntegrationProvider,
  useGithubIntegration,
} from '@/next/hooks/use-github-integration';
import useCan from '@/next/hooks/use-can';
import { github } from '@/next/lib/can/features/github.permissions';
import {
  Alert,
  AlertTitle,
  AlertDescription,
} from '@/next/components/ui/alert';
import { Lock } from 'lucide-react';

function GithubInstallationsEmptyState() {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-8">
      <p className="text-md">No GitHub installations found.</p>
      <p className="text-sm text-muted-foreground">
        Connect your GitHub account to get started.
      </p>
      <a href="/api/v1/cloud/users/github-app/start">
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Connect GitHub Account
        </Button>
      </a>
    </div>
  );
}

function GithubInstallationsList() {
  const { installations, linkInstallation } = useGithubIntegration();
  const [installationToLink, setInstallationToLink] = useState<
    string | undefined
  >();

  const cols = githubInstallationsColumns((installationId: string) => {
    setInstallationToLink(installationId);
  });

  return (
    <>
      <DataTable
        isLoading={installations.isLoading}
        columns={cols}
        data={installations.data || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
        emptyState={<GithubInstallationsEmptyState />}
      />
      <Dialog
        open={installationToLink !== undefined}
        onOpenChange={(open) => !open && setInstallationToLink(undefined)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Are you sure?</DialogTitle>
            <DialogDescription>
              Linking this app will allow other members of the tenant to view
              this installation. Users will only be able to deploy to
              repositories that they have access to.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setInstallationToLink(undefined)}
            >
              Cancel
            </Button>
            <Button
              onClick={async () => {
                if (installationToLink) {
                  await linkInstallation.mutateAsync(installationToLink);
                  setInstallationToLink(undefined);
                }
              }}
              loading={linkInstallation.isPending}
            >
              Yes, link to tenant
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export default function GithubPage() {
  const { canWithReason } = useCan();
  const { allowed: canManage, message: canManageMessage } = canWithReason(
    github.manage(),
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Link your Github account to integrate with CI/CD to deploy tasks and workflows">
          Github Integration
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <a href="/api/v1/cloud/users/github-app/start">
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                Connect new account
              </Button>
            </a>
            <></>
            {/* <DocsButton doc={docs.home['github-integration']} size="icon" /> */}
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      {canManageMessage && (
        <Alert variant="warning">
          <Lock className="w-4 h-4 mr-2" />
          <AlertTitle>Role required</AlertTitle>
          <AlertDescription>{canManageMessage}</AlertDescription>
        </Alert>
      )}

      {canManage && (
        <>
          <Separator className="my-4" />
          <GithubIntegrationProvider>
            <GithubInstallationsList />
          </GithubIntegrationProvider>
        </>
      )}
    </BasicLayout>
  );
}
