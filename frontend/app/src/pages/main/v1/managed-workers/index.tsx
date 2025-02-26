import { Separator } from '@/components/v1/ui/separator';
import invariant from 'tiny-invariant';
import { Link, useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { Button } from '@/components/v1/ui/button';

export default function ManagedWorkers() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Managed Compute
          </h2>
          <Link to="/managed-workers/create">
            <Button>Deploy Workers</Button>
          </Link>
        </div>
        <Separator className="my-4" />
        <ManagedWorkersTable />
      </div>
    </div>
  );
}
