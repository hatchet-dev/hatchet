import { Separator } from '@/components/ui/separator';
import { queries, Worker } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { Link, useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { relativeDate } from '@/lib/utils';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { columns } from './components/step-runs-columns';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import { Badge } from '@/components/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

export const isHealthy = (worker?: Worker) => {
  const reasons = [];

  if (!worker) {
    reasons.push('Worker is undefined');
    return reasons;
  }

  if (worker.status !== 'ACTIVE') {
    reasons.push('Worker is not active');
  }

  if (!worker.dispatcherId) {
    reasons.push('Worker has no assigned dispatcher');
  }

  if (!worker.lastHeartbeatAt) {
    reasons.push('Worker has no heartbeat');
  } else {
    const beat = new Date(worker.lastHeartbeatAt).getTime();
    const now = new Date().getTime();

    if (now - beat > 6 * 1000) {
      reasons.push('Worker has missed a heartbeat');
    }
  }

  return reasons;
};

export const WorkerStatus = ({
  status,
  health,
}: {
  status?: 'ACTIVE' | 'INACTIVE';
  health: string[];
}) => (
  <div className="flex flex-row gap-2 item-center">
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <Badge variant={health.length === 0 ? 'successful' : 'failed'}>
            {health.length === 0 ? 'Healthy' : 'Unhealthy'}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          {health.map((reason, i) => (
            <div key={i}>{reason}</div>
          ))}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
    <span className="py-0.5">
      <Badge variant={status === 'ACTIVE' ? 'successful' : 'failed'}>
        {status === 'ACTIVE' ? 'Active' : 'Inactive'}
      </Badge>
    </span>
  </div>
);

export default function ExpandedWorkflowRun() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.worker);

  const workerQuery = useQuery({
    ...queries.workers.get(params.worker),
    refetchInterval: 5000,
  });

  const worker = workerQuery.data;

  const healthy = isHealthy(worker);

  if (!worker || workerQuery.isLoading || !workerQuery.data) {
    return <Loading />;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              <Link to="/workers">Workers/</Link>
              {worker.name}
            </h2>
          </div>
          <WorkerStatus status={worker.status} health={healthy} />
        </div>
        <Separator className="my-4" />
        <p className="mt-1 max-w-2xl text-gray-700 dark:text-gray-300">
          Started {relativeDate(worker.metadata?.createdAt)}
          <br />
          Last seen {relativeDate(worker?.lastHeartbeatAt)} <br />
          {(worker.maxRuns ?? 0) > 0
            ? `${worker.availableRuns} / ${worker.maxRuns ?? 0}`
            : '∞'}{' '}
          available run slots
        </p>
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
