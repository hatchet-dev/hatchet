import { useResourceLimitColumns } from './resource-limit-columns';
import { DocsButton } from '@/components/v1/docs/docs-button';
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
        <p className="text-sm text-muted-foreground">
          No resource limits configured for this tenant.
          {showDocsOnEmpty ? (
            <>
              {' '}
              Set <code>SERVER_ENFORCE_LIMITS</code> to <code>true</code> in
              your environment variables to enable resource limits.
            </>
          ) : null}
        </p>
      )}

      {showDocsOnEmpty && resourceLimits.length === 0 ? (
        <div className="mt-4">
          <DocsButton
            doc={docsPages['self-hosting']['configuration-options']}
            label="Learn about engine configuration options"
          />
        </div>
      ) : null}
    </section>
  );
}
