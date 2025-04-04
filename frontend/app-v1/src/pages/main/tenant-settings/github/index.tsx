import { Separator } from '@/components/v1/ui/separator';
import { useMutation, useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

import { columns as githubInstallationsColumns } from './components/github-installations-columns';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { Button } from '@/components/v1/ui/button';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useTenant } from '@/lib/atoms';
import invariant from 'tiny-invariant';
import { useState } from 'react';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';

export default function Github() {
  const cloudMeta = useCloudApiMeta();

  const hasGithubIntegration = cloudMeta?.data.canLinkGithub;

  if (!cloudMeta || !hasGithubIntegration) {
    return (
      <div className="flex-grow h-full w-full">
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Not enabled for this tenant or instance.
        </p>
      </div>
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Github Integration
        </h2>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Link your Github account to Hatchet to integrate with CI/CD and
          workflow versioning.
        </p>
        <Separator className="my-4" />
        {hasGithubIntegration && <Separator className="my-4" />}
        {hasGithubIntegration && <GithubInstallationsList />}
      </div>
    </div>
  );
}

function GithubInstallationsList() {
  const tenant = useTenant();
  invariant(tenant.tenantId, 'tenant should be set');

  const [installationToLink, setInstallationToLink] = useState<
    string | undefined
  >();

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations(tenant.tenantId),
  });

  const { handleApiError } = useApiError({});

  const linkInstallationToTenantMutation = useMutation({
    mutationKey: [
      'github-app:update:installation',
      tenant.tenantId,
      installationToLink,
    ],
    mutationFn: async () => {
      invariant(installationToLink, 'installationToLink should be set');
      const res = await cloudApi.githubAppUpdateInstallation(
        installationToLink,
        {
          tenant: tenant.tenantId,
        },
      );
      return res.data;
    },
    onSuccess: () => {
      setInstallationToLink(undefined);
      listInstallationsQuery.refetch();
    },
    onError: handleApiError,
  });

  const cols = githubInstallationsColumns((installationId: string) => {
    setInstallationToLink(installationId);
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Github Accounts
        </h3>
        <a href="/api/v1/cloud/users/github-app/start">
          <Button key="create-api-token">Link new account</Button>
        </a>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={listInstallationsQuery.isLoading}
        columns={cols}
        data={listInstallationsQuery.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      <ConfirmDialog
        title={`Are you sure?`}
        description={`Linking this app to ${tenant.tenant.name} will allow other members of the tenant to view this installation. Users will only be able to deploy to repositories that they have access to.`}
        submitLabel={'Yes, link to tenant'}
        submitVariant={'default'}
        onSubmit={linkInstallationToTenantMutation.mutate}
        onCancel={function (): void {
          setInstallationToLink(undefined);
        }}
        isLoading={linkInstallationToTenantMutation.isPending}
        isOpen={installationToLink !== undefined}
      />
    </div>
  );
}
