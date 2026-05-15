import { SettingsPageHeader } from '../components/settings-page-header';
import { resourceLimitColumns } from './components/resource-limit-columns';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Spinner } from '@/components/v1/ui/loading';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useAppContext } from '@/providers/app-context';
import { useQuery } from '@tanstack/react-query';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();
  const appContext = useAppContext();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const resourceLimits = resourcePolicyQuery.data?.limits || [];

  if (resourcePolicyQuery.isLoading || !appContext.isUserUniverseLoaded) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      <SettingsPageHeader
        title="Resource limits"
        description="Review curently configured resource limits for this tenant. Once limits are exceeded, requests will be rejected."
      />
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        {resourceLimits.length > 0 ? (
          <SimpleTable
            columns={resourceLimitColumns}
            data={resourceLimits}
            rowKey={(row) => row.metadata.id}
          />
        ) : (
          <div className="flex flex-col items-center gap-y-4 py-8 text-center text-sm text-muted-foreground">
            <p>No resource limits configured.</p>
            <DocsButton
              doc={docsPages.v1['rate-limits']}
              label="Learn about resource limits"
            />
          </div>
        )}
      </div>
    </div>
  );
}
