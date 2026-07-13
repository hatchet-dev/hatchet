import { useResourceLimitColumns } from './resource-limit-columns';
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Spinner } from '@/components/v1/ui/loading';
import useCloud from '@/hooks/use-cloud';
import { queries } from '@/lib/api';
import type { TenantResourceLimit } from '@/lib/api';
import { docsPages } from '@/lib/generated/docs';
import { useQuery } from '@tanstack/react-query';

const BILLING_SYNC_REFETCH_INTERVAL_MS = 5000;

type TenantResourceLimitsTableProps = {
  tenantId: string;
  tenantName?: string;
  limits?: TenantResourceLimit[];
  isLoading?: boolean;
  showDocsOnEmpty?: boolean;
};

export function TenantResourceLimitsTable({
  tenantId,
  tenantName,
  limits,
  isLoading,
  showDocsOnEmpty = false,
}: TenantResourceLimitsTableProps) {
  const { isCloudEnabled } = useCloud();
  const resourceLimitColumns = useResourceLimitColumns();

  // The SERVER_ENFORCE_LIMITS hint and engine-configuration docs only apply to
  // self-hosted deployments — on cloud, limits come from the billing plan.
  const showSelfHostDocs = showDocsOnEmpty && !isCloudEnabled;

  const billingSyncRefetchInterval = isCloudEnabled
    ? BILLING_SYNC_REFETCH_INTERVAL_MS
    : false;

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
    enabled: limits === undefined,
    refetchInterval: billingSyncRefetchInterval,
  });

  const resourceLimits = limits ?? resourcePolicyQuery.data?.limits ?? [];

  if (isLoading ?? resourcePolicyQuery.isLoading) {
    return (
      <div className="py-6">
        <Spinner />
      </div>
    );
  }

  return (
    <section>
      {tenantName ? (
        <h3 className="mb-4 text-base font-semibold text-foreground">
          {tenantName}
        </h3>
      ) : null}

      {resourceLimits.length > 0 ? (
        <SimpleTable
          columns={resourceLimitColumns}
          data={resourceLimits}
          rowKey={(row) => row.metadata.id}
        />
      ) : (
        <EmptyState
          title="No resource limits configured"
          description={
            showSelfHostDocs
              ? 'Resource limits cap the number of active runs and tasks in your tenant. Set SERVER_ENFORCE_LIMITS=true to enable them.'
              : 'Resource limits cap the number of active runs and tasks in your tenant.'
          }
          docPage={
            showSelfHostDocs
              ? docsPages['self-hosting']['configuration-options']
              : undefined
          }
          docLabel={
            showSelfHostDocs
              ? 'Learn about engine configuration options'
              : undefined
          }
        />
      )}
    </section>
  );
}
