import { Separator } from '@/components/v1/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './components/resource-limit-columns';
import { PaymentMethods, Subscription } from '@/components/v1/cloud/billing';
import { Spinner } from '@/components/v1/ui/loading';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';

export default function ResourceLimits() {
  const { tenant } = useOutletContext<TenantContextType>();
  const cloudMeta = useCloudApiMeta();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenant.metadata.id),
  });

  const billingState = useQuery({
    ...queries.cloud.billing(tenant.metadata.id),
    enabled: !!cloudMeta?.data.canBill,
  });

  const cols = columns();

  const billingEnabled = cloudMeta?.data.canBill;

  const hasPaymentMethods =
    (billingState.data?.paymentMethods?.length || 0) > 0;

  if (resourcePolicyQuery.isLoading || billingState.isLoading) {
    return (
      <div className="flex-grow h-full w-full px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      {billingEnabled && (
        <>
          <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
            <div className="flex flex-row justify-between items-center">
              <h2 className="text-2xl font-semibold leading-tight text-foreground">
                Billing and Limits
              </h2>
            </div>
          </div>
          <Separator className="my-4" />
          <PaymentMethods
            hasMethods={hasPaymentMethods}
            methods={billingState.data?.paymentMethods}
          />
          <Separator className="my-4" />
          <Subscription
            hasPaymentMethods={hasPaymentMethods}
            active={billingState.data?.subscription}
            plans={billingState.data?.plans}
            coupons={billingState.data?.coupons}
          />
          <Separator className="my-4" />
        </>
      )}

      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h3 className="text-xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h3>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
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
          filters={[]}
          getRowId={(row) => row.metadata.id}
        />
      </div>
    </div>
  );
}
