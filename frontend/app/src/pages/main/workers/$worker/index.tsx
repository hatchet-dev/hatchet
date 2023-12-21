import { Separator } from '@/components/ui/separator';
import { queries } from '@/lib/api';
import { currTenantAtom } from '@/lib/atoms';
import { useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import { useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { relativeDate } from '@/lib/utils';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './components/step-runs-columns';
import { Loading } from '@/components/ui/loading.tsx';

export default function ExpandedWorkflowRun() {
  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const params = useParams();
  invariant(params.worker);

  const workerQuery = useQuery({
    ...queries.workers.get(params.worker),
  });

  if (workerQuery.isLoading || !workerQuery.data) {
    return <Loading />;
  }

  const worker = workerQuery.data;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {worker.name}
            </h2>
            <div className="text-sm text-gray-500">
              Last seen {relativeDate(worker.lastHeartbeatAt)}
            </div>
          </div>
        </div>
        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
          Recent Step Runs
        </h3>
        <DataTable
          columns={columns}
          data={worker.recentStepRuns || []}
          filters={[]}
        />
        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
          Worker Actions
        </h3>
        <div className="flex-wrap flex flex-row gap-4">
          {worker.actions?.map((action) => {
            return (
              <Button variant="outline" key={action}>
                {action}
              </Button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
