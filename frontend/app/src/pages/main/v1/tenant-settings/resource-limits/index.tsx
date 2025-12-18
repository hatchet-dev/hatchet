import { columns } from './components/resource-limit-columns';
import { Subscription } from '@/components/v1/cloud/billing';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useQuery } from '@tanstack/react-query';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();

  const { data: cloudMeta } = useCloudApiMeta();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const billingState = useQuery({
    ...queries.cloud.billing(tenantId),
    enabled: !!tenantId && !!cloudMeta?.data.canBill,
    retry: false,
  });

  const cols = columns();

  const billingEnabled = cloudMeta?.data.canBill;

  if (resourcePolicyQuery.isLoading || billingState.isLoading) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      {billingEnabled && (
        <>
          <Subscription
            active={billingState.data?.currentSubscription}
            upcoming={billingState.data?.upcomingSubscription}
            plans={billingState.data?.plans}
            coupons={billingState.data?.coupons}
          />
          <Separator className="my-4" />
        </>
      )}

      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <h3 className="text-xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h3>
        </div>
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Resource limits are used to control the usage of resources within a
          tenant. When a limit is reached, the system will take action based on
          the limit type. Please upgrade your plan, or{' '}
          <a href="https://hatchet.run/office-hours" className="underline">
            contact us
          </a>{' '}
          if you need to adjust your limits.
        </p>

        <DataTable
          isLoading={resourcePolicyQuery.isLoading}
          columns={cols}
          data={resourcePolicyQuery.data?.limits || []}
          getRowId={(row) => row.metadata.id}
        />
      </div>
    </div>
  );
}
