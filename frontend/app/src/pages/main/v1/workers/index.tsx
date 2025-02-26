import { Separator } from '@/components/v1/ui/separator';
import invariant from 'tiny-invariant';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';

import { WorkersTable } from './components/worker-table';
import { ServerStackIcon } from '@heroicons/react/24/outline';

export default function Workers() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              Workers
            </h2>
          </div>
        </div>
        <Separator className="my-4" />
        <WorkersTable />
      </div>
    </div>
  );
}
