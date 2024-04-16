import { Separator } from '@/components/ui/separator';
import invariant from 'tiny-invariant';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';

import { WorkersTable } from './components/worker-table';

export default function Workers() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workers
        </h2>
        <Separator className="my-4" />
        <WorkersTable />
      </div>
    </div>
  );
}
