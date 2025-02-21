import { Separator } from '@/components/v1/ui/separator';
import api, { queries, UpdateWorkerRequest, Worker } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { ArrowPathIcon, ServerStackIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
import { BiDotsVertical } from 'react-icons/bi';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useState } from 'react';
import { RecentWebhookRequests } from '../webhooks/components/recent-webhook-requests';
import { WorkflowRunsTable } from '../../workflow-runs/components/workflow-runs-table';
export const isHealthy = (worker?: Worker) => {
  const reasons = [];

  if (!worker) {
    reasons.push('Worker is undefined');
    return reasons;
  }

  if (worker.status !== 'ACTIVE') {
    reasons.push('Worker has stopped heartbeating');
  }

  if (!worker.dispatcherId) {
    reasons.push('Worker has no assigned dispatcher');
  }

  if (!worker.lastHeartbeatAt) {
    reasons.push('Worker has no heartbeat');
  }

  return reasons;
};

export const WorkerStatus = ({
  status = 'INACTIVE',
  health,
}: {
  status?: 'ACTIVE' | 'INACTIVE' | 'PAUSED';
  health: string[];
}) => {
  const label: Record<typeof status, string> = {
    ACTIVE: 'Active',
    INACTIVE: 'Inactive',
    PAUSED: 'Paused',
  };

  const variant: Record<typeof status, BadgeProps['variant']> = {
    ACTIVE: 'successful',
    INACTIVE: 'failed',
    PAUSED: 'inProgress',
  };

  return (
    <div className="flex flex-row gap-2 item-center">
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger>
            <Badge variant={variant[status]}>{label[status]}</Badge>
          </TooltipTrigger>
          <TooltipContent>
            {health.map((reason, i) => (
              <div key={i}>{reason}</div>
            ))}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    </div>
  );
};

export default function ExpandedWorkflowRun() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const { handleApiError } = useApiError({});

  const params = useParams();
  invariant(params.worker);

  const workerQuery = useQuery({
    ...queries.workers.get(params.worker),
    refetchInterval: 3000,
  });

  const [rotate, setRotate] = useState(false);

  const worker = workerQuery.data;

  const healthy = isHealthy(worker);

  const updateWorker = useMutation({
    mutationKey: ['worker:update', worker?.metadata.id],
    mutationFn: async (data: UpdateWorkerRequest) =>
      (await api.workerUpdate(worker!.metadata.id, data)).data,
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.workers.get(worker!.metadata.id).queryKey,
      });
    },
    onError: handleApiError,
  });

  if (!worker || workerQuery.isLoading || !workerQuery.data) {
    return <Loading />;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <Badge>{worker.type}</Badge>
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              <Link to="/workers">Workers/</Link>
              {worker.webhookUrl || worker.name}
            </h2>
          </div>
          <div className="flex flex-row gap-2">
            <WorkerStatus status={worker.status} health={healthy} />
            <DropdownMenu>
              <DropdownMenuTrigger>
                <Button
                  aria-label="Workflow Actions"
                  size="icon"
                  variant="ghost"
                >
                  <BiDotsVertical />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent>
                <DropdownMenuItem
                  disabled={worker.status === 'INACTIVE'}
                  onClick={() => {
                    updateWorker.mutate({
                      isPaused: worker.status === 'PAUSED' ? false : true,
                    });
                  }}
                >
                  {worker.status === 'PAUSED' ? 'Resume' : 'Pause'} Step Run
                  Assignment
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
        <Separator className="my-4" />
        <p className="mt-1 max-w-2xl text-gray-700 dark:text-gray-300">
          First Connected: <RelativeDate date={worker.metadata?.createdAt} />
          {worker.lastListenerEstablished && (
            <>
              <br />
              Last Listener Established:{' '}
              <RelativeDate date={worker.lastListenerEstablished} />
            </>
          )}
          <br />
          Last Heartbeat:{' '}
          {worker.lastHeartbeatAt ? (
            <RelativeDate date={worker.lastHeartbeatAt} />
          ) : (
            'never'
          )}
          <br />
        </p>
        <Separator className="my-4" />

        <div className="flex flex-row justify-between items-center mb-4">
          <h3 className="text-xl font-bold leading-tight text-foreground">
            {(worker.maxRuns ?? 0) > 0
              ? `${worker.availableRuns} / ${worker.maxRuns ?? 0}`
              : '100'}{' '}
            Available Run Slots
          </h3>

          <Button
            size="icon"
            aria-label="Refresh"
            variant="outline"
            disabled={workerQuery.isFetching}
            onClick={() => {
              workerQuery.refetch();
              setRotate(!rotate);
            }}
          >
            <ArrowPathIcon
              className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
            />
          </Button>
        </div>
        <div className="mb-4 text-sm text-gray-700 dark:text-gray-300">
          A slot represents one step run on a worker to limit load.{' '}
          <a
            href="https://docs.hatchet.run/sdks/python-sdk/worker"
            className="underline"
          >
            Learn more.
          </a>
        </div>

        {/* <WorkerSlotGrid slots={worker.slots} /> */}

        <Separator className="my-4" />
        <div className="flex flex-row justify-between items-center mb-4">
          <h3 className="text-xl font-bold leading-tight text-foreground">
            Recent Tasks
          </h3>
        </div>
        <WorkflowRunsTable
          workerId={worker.metadata.id}
          createdAfter={worker.metadata.createdAt}
          showMetrics={false}
          showCounts={false}
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
        {worker.webhookId && (
          <>
            <Separator className="my-4" />
            <div className="flex flex-row justify-between items-center mb-4">
              <h3 className="text-xl font-bold leading-tight text-foreground">
                Recent HTTP Health Checks
              </h3>
            </div>
            <RecentWebhookRequests webhookId={worker.webhookId} />
          </>
        )}

        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
          Worker Labels
        </h3>
        <div className="mb-4 text-sm text-gray-700 dark:text-gray-300">
          Worker labels are key-value pairs that can be used to prioritize
          assignment of steps to specific workers.{' '}
          <a
            className="underline"
            href="https://docs.hatchet.run/home/features/worker-assignment/worker-affinity#specifying-worker-labels"
          >
            Learn more.
          </a>
        </div>
        <div className="flex gap-2">
          {!worker.labels || worker.labels.length === 0 ? (
            <>
              <>No Labels Assigned.</>
            </>
          ) : (
            worker.labels?.map(({ key, value }) => (
              <Badge key={key}>
                {key}:{value}
              </Badge>
            ))
          )}
        </div>
        {worker.runtimeInfo && (
          <>
            <Separator className="my-4" />
            <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
              Worker Runtime Info
            </h3>
            <div className="mb-4 text-sm text-gray-700 dark:text-gray-300">
              {worker.runtimeInfo?.sdkVersion && (
                <div>
                  <b>Hatchet SDK</b>: {worker.runtimeInfo?.sdkVersion}
                </div>
              )}
              {worker.runtimeInfo?.languageVersion && (
                <div>
                  <b>Runtime</b>: {worker.runtimeInfo?.language}{' '}
                  {worker.runtimeInfo?.languageVersion}
                </div>
              )}
              {worker.runtimeInfo?.os && (
                <div>
                  <b>OS</b>: {worker.runtimeInfo?.os}
                </div>
              )}
              {worker.runtimeInfo?.runtimeExtra && (
                <div>
                  <b>Runtime Extra</b>: {worker.runtimeInfo?.runtimeExtra}
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
