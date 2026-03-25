import {
  GithubAccountCell,
  GithubLinkCell,
  GithubSettingsCell,
} from './components/github-installations-columns';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { GithubAppInstallation } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState, useMemo } from 'react';
import invariant from 'tiny-invariant';

export default function Github() {
  const { cloud } = useCloud();

  const hasGithubIntegration = cloud && cloud.canLinkGithub;

  if (!cloud || !hasGithubIntegration) {
    return (
      <div className="h-full w-full flex-grow">
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Not enabled for this tenant or instance.
        </p>
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Github Integration
        </h2>
        <p className="my-4 text-gray-700 dark:text-gray-300">
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
  const { tenantId } = useCurrentTenantId();
  const { tenant } = useTenantDetails();

  const [installationToLink, setInstallationToLink] = useState<
    string | undefined
  >();

  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations(tenantId),
  });

  const { handleApiError } = useApiError({});

  const linkInstallationToTenantMutation = useMutation({
    mutationKey: [
      'github-app:update:installation',
      tenantId,
      installationToLink,
    ],
    mutationFn: async () => {
      invariant(installationToLink, 'installationToLink should be set');
      const res = await cloudApi.githubAppUpdateInstallation(
        installationToLink,
        {
          tenant: tenantId,
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

  const githubColumns = useMemo(
    () => [
      {
        columnLabel: 'Account name',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubAccountCell installation={installation} />
        ),
      },
      {
        columnLabel: 'Link to tenant?',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubLinkCell
            installation={installation}
            onLinkToTenant={(installationId: string) => {
              setInstallationToLink(installationId);
            }}
          />
        ),
      },
      {
        columnLabel: 'Github Settings',
        cellRenderer: (installation: GithubAppInstallation) => (
          <GithubSettingsCell installation={installation} />
        ),
      },
    ],
    [],
  );

  const currentPath = window.location.pathname;

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Github Accounts
        </h3>
        <a
          href={`/api/v1/cloud/users/github-app/start?redirect_to=${encodeURIComponent(currentPath)}&with_repo_installation=false`}
        >
          <Button key="create-api-token">Link new account</Button>
        </a>
      </div>
      <Separator className="my-4" />
      {(listInstallationsQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={githubColumns}
          data={listInstallationsQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No Github accounts linked. Link an account to integrate with CI/CD.
        </div>
      )}
      <ConfirmDialog
        title={`Are you sure?`}
        description={`Linking this app to ${tenant?.name} will allow other members of the tenant to view this installation. Users will only be able to deploy to repositories that they have access to.`}
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
