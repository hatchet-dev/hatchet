import { Separator } from '@/components/ui/separator';
import { useApiMetaIntegrations } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

import { columns as githubInstallationsColumns } from './components/github-installations-columns';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { Button } from '@/components/ui/button';

export default function Github() {
  const integrations = useApiMetaIntegrations();

  const hasGithubIntegration = integrations?.find((i) => i.name === 'github');

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
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
  const listInstallationsQuery = useQuery({
    ...queries.github.listInstallations,
  });

  const cols = githubInstallationsColumns();

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Github Accounts
        </h3>
        <a href="/api/v1/users/github/start">
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
    </div>
  );
}
