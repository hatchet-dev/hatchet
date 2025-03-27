import { Separator } from '@/components/ui/separator';
import { Link } from 'react-router-dom';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { Button } from '@/components/ui/button';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';

export default function ManagedWorkers() {
  const meta = useCloudApiMeta();

  const requireBillingForManagedCompute =
    meta?.data.requireBillingForManagedCompute;

  if (requireBillingForManagedCompute) {
    return <div>no billing enabled.</div>;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
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
