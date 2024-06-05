import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './components/resource-limit-columns';

export default function ResourceLimits() {
  const { tenant } = useOutletContext<TenantContextType>();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenant.metadata.id),
  });

  const cols = columns();

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h2>
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
        <Separator className="my-4" />
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
