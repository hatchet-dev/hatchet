import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './components/resource-limit-columns';
import PaymentMethods from './components/payment-methods';
import useApiMeta from '@/pages/auth/hooks/use-api-meta';

export default function ResourceLimits() {
  const { tenant } = useOutletContext<TenantContextType>();
  const meta = useApiMeta();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenant.metadata.id),
  });

  const cols = columns();

  const billingEnabled = meta.data?.data.billing;

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
            manageLink={resourcePolicyQuery.data?.checkoutLink}
            methods={resourcePolicyQuery.data?.paymentMethods}
          />
          <Separator className="my-4" />
          <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
            <div className="flex flex-row justify-between items-center">
              <h3 className="text-xl font-semibold leading-tight text-foreground">
                Subscription
              </h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 my-4"></p>
            {JSON.stringify(resourcePolicyQuery.data?.subscription)}
          </div>
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
          the limit type. Please{' '}
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
