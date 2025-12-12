import { Separator } from '@/components/v1/ui/separator';
import api, { queries, UpdateWorkerRequest, Worker } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import invariant from 'tiny-invariant';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
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
import { RecentWebhookRequests } from '../webhooks/components/recent-webhook-requests';
import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { capitalize } from '@/lib/utils';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { flattenDAGsKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { useMemo, useState } from 'react';
import { appRoutes } from '@/router';
const isHealthy = (worker?: Worker) => {
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

const WorkerStatus = ({
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

const N_ACTIONS_TO_PREVIEW = 10;

export default function ExpandedWorkflowRun() {
  const { handleApiError } = useApiError({});
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const [showAllActions, setShowAllActions] = useState(false);

  const params = useParams({ from: appRoutes.tenantWorkerRoute.to });
  invariant(params.worker);

  const workerQuery = useQuery({
    ...queries.workers.get(params.worker),
    refetchInterval,
  });

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

  const registeredWorkflows = useMemo(
    () => worker?.registeredWorkflows || [],
    [worker],
  );

  const filteredWorkflows = useMemo(() => {
    if (showAllActions) {
      return registeredWorkflows;
    }

    return registeredWorkflows.slice(0, N_ACTIONS_TO_PREVIEW);
  }, [showAllActions, registeredWorkflows]);

  if (!worker || workerQuery.isLoading || !workerQuery.data) {
    return <Loading />;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto px-4 sm:px-6 lg:px-8 flex flex-col">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center justify-between">
            <ServerStackIcon className="h-6 w-6 text-foreground mt-1" />
            <Badge>{worker.type}</Badge>
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              <Link
                to={appRoutes.tenantWorkersRoute.to}
                params={{ tenant: tenantId }}
              >
                Workers/
              </Link>
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
        </div>
        <div className="mb-4 text-sm text-gray-700 dark:text-gray-300">
          A slot represents one task run on a worker to limit load.{' '}
          <a href="https://docs.hatchet.run/home/workers" className="underline">
            Learn more.
          </a>
        </div>

        <Separator className="my-4" />
        <div className="flex flex-row justify-between items-center mb-4">
          <h3 className="text-xl font-bold leading-tight text-foreground">
            Recent Task Runs
          </h3>
        </div>
        <RunsProvider
          tableKey={`worker-${worker.metadata.id}`}
          display={{
            hideMetrics: true,
            hideCounts: true,
            hideTriggerRunButton: true,
            hiddenFilters: [flattenDAGsKey],
            hideCancelAndReplayButtons: true,
          }}
          runFilters={{
            workerId: worker.metadata.id,
          }}
        >
          <RunsTable />
        </RunsProvider>
        <Separator className="my-4" />
        <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
          Registered Workflows
        </h3>
        <div className="flex-wrap flex flex-row gap-4">
          {filteredWorkflows.map((workflow) => {
            return (
              <Link
                to={appRoutes.tenantWorkflowRoute.to}
                params={{ tenant: tenantId, workflow: workflow.id }}
                key={workflow.id}
              >
                <Button variant="outline">{workflow.name}</Button>
              </Link>
            );
          })}
        </div>
        <div className="flex flex-row w-full items-center justify-center py-4">
          {!showAllActions &&
            registeredWorkflows.length > N_ACTIONS_TO_PREVIEW && (
              <Button variant="outline" onClick={() => setShowAllActions(true)}>
                {`Show All (${registeredWorkflows.length - N_ACTIONS_TO_PREVIEW} more)`}
              </Button>
            )}
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

        {worker.labels && worker.labels.length > 0 && (
          <>
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
              {worker.labels?.map(({ key, value }) => (
                <Badge key={key}>
                  {key}:{value}
                </Badge>
              ))}
            </div>
          </>
        )}
        {worker.runtimeInfo &&
          (worker.runtimeInfo?.sdkVersion ||
            worker.runtimeInfo?.languageVersion ||
            worker.runtimeInfo?.os ||
            worker.runtimeInfo?.runtimeExtra) && (
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
                    <b>Runtime</b>:{' '}
                    {capitalize(worker.runtimeInfo?.language ?? '')}{' '}
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
