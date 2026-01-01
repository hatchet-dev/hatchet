import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { flattenDAGsKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge, BadgeProps } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Loading } from '@/components/v1/ui/loading.tsx';
import {
  PortalTooltip,
  PortalTooltipContent,
  PortalTooltipProvider,
  PortalTooltipTrigger,
} from '@/components/v1/ui/portal-tooltip';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { queries, UpdateWorkerRequest, Worker } from '@/lib/api';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { useApiError } from '@/lib/hooks';
import { capitalize } from '@/lib/utils';
import { ResourceNotFound } from '@/pages/error/components/resource-not-found';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { ServerStackIcon } from '@heroicons/react/24/outline';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import { isAxiosError } from 'axios';
import { useMemo, useState } from 'react';
import { BiDotsVertical } from 'react-icons/bi';

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
    <div className="item-center flex flex-row gap-2">
      <PortalTooltipProvider>
        <PortalTooltip>
          <PortalTooltipTrigger>
            <Badge variant={variant[status]}>{label[status]}</Badge>
          </PortalTooltipTrigger>
          <PortalTooltipContent>
            {health.map((reason, i) => (
              <div key={i}>{reason}</div>
            ))}
          </PortalTooltipContent>
        </PortalTooltip>
      </PortalTooltipProvider>
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

  const workerQuery = useQuery({
    ...queries.workers.get(params.worker),
    refetchInterval,
    retry: (_failureCount, error) => shouldRetryQueryError(error),
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

  if (workerQuery.isLoading) {
    return <Loading />;
  }

  if (workerQuery.isError) {
    if (
      isAxiosError(workerQuery.error) &&
      workerQuery.error.response?.status === 404
    ) {
      return (
        <ResourceNotFound
          resource="Worker"
          primaryAction={{
            label: 'Back to Workers',
            navigate: {
              to: appRoutes.tenantWorkersRoute.to,
              params: { tenant: tenantId },
            },
          }}
        />
      );
    }

    throw workerQuery.error;
  }

  if (!worker) {
    return <Loading />;
  }

  const availableSlots = worker.availableRuns ?? 0;
  const maxSlots = worker.maxRuns ?? 0;
  const slotPercentage =
    maxSlots > 0 ? Math.round((availableSlots / maxSlots) * 100) : 100;

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto flex flex-col gap-6 px-4 py-6 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="flex flex-row items-center justify-between">
          <div className="flex flex-row items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg border bg-muted/50">
              <ServerStackIcon className="h-6 w-6 text-foreground" />
            </div>
            <div>
              <div className="text-sm text-muted-foreground">
                <Link
                  to={appRoutes.tenantWorkersRoute.to}
                  params={{ tenant: tenantId }}
                  className="hover:underline"
                >
                  Workers
                </Link>
              </div>
              <h1 className="text-2xl font-bold leading-tight text-foreground">
                {worker.webhookUrl || worker.name}
              </h1>
            </div>
          </div>
          <div className="flex flex-row gap-2">
            <WorkerStatus status={worker.status} health={healthy} />
            <DropdownMenu>
              <DropdownMenuTrigger>
                <Button aria-label="Worker Actions" size="icon" variant="ghost">
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

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-3">
          {/* Connection Info Card */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Connection Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div>
                <div className="text-muted-foreground">First Connected</div>
                <div className="font-medium">
                  <RelativeDate date={worker.metadata?.createdAt} />
                </div>
              </div>
              {worker.lastListenerEstablished && (
                <div>
                  <div className="text-muted-foreground">
                    Last Listener Established
                  </div>
                  <div className="font-medium">
                    <RelativeDate date={worker.lastListenerEstablished} />
                  </div>
                </div>
              )}
              <div>
                <div className="text-muted-foreground">Last Heartbeat</div>
                <div className="font-medium">
                  {worker.lastHeartbeatAt ? (
                    <RelativeDate date={worker.lastHeartbeatAt} />
                  ) : (
                    'Never'
                  )}
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Available Slots Card */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Available Run Slots</CardTitle>
              <CardDescription>
                Slots represent concurrent task runs on this worker
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex items-baseline justify-between">
                  <span className="text-3xl font-bold">
                    {maxSlots > 0 ? availableSlots : '∞'}
                  </span>
                  {maxSlots > 0 && (
                    <span className="text-sm text-muted-foreground">
                      / {maxSlots} total
                    </span>
                  )}
                </div>
                {maxSlots > 0 && (
                  <div className="space-y-1">
                    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full bg-primary transition-all"
                        style={{ width: `${slotPercentage}%` }}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {slotPercentage}% available
                    </div>
                  </div>
                )}
              </div>
              <div className="mt-4">
                <a
                  href="https://docs.hatchet.run/home/workers"
                  className="text-xs text-muted-foreground underline hover:text-foreground"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Learn more about slots
                </a>
              </div>
            </CardContent>
          </Card>

          {/* Runtime Info Card */}
          {worker.runtimeInfo &&
            (worker.runtimeInfo?.sdkVersion ||
              worker.runtimeInfo?.languageVersion ||
              worker.runtimeInfo?.os) && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Runtime Info</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  {worker.runtimeInfo?.sdkVersion && (
                    <div>
                      <div className="text-muted-foreground">Hatchet SDK</div>
                      <div className="font-medium">
                        {worker.runtimeInfo.sdkVersion}
                      </div>
                    </div>
                  )}
                  {worker.runtimeInfo?.languageVersion && (
                    <div>
                      <div className="text-muted-foreground">Runtime</div>
                      <div className="font-medium">
                        {capitalize(worker.runtimeInfo.language ?? '')}{' '}
                        {worker.runtimeInfo.languageVersion}
                      </div>
                    </div>
                  )}
                  {worker.runtimeInfo?.os && (
                    <div>
                      <div className="text-muted-foreground">OS</div>
                      <div className="font-medium">{worker.runtimeInfo.os}</div>
                    </div>
                  )}
                  {worker.runtimeInfo?.runtimeExtra && (
                    <div>
                      <div className="text-muted-foreground">Runtime Extra</div>
                      <div className="font-medium">
                        {worker.runtimeInfo.runtimeExtra}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}
        </div>

        {/* Worker Labels */}
        {worker.labels && worker.labels.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Worker Labels</CardTitle>
              <CardDescription>
                Key-value pairs used to prioritize step assignment to specific
                workers.{' '}
                <a
                  className="underline hover:text-foreground"
                  href="https://docs.hatchet.run/home/features/worker-assignment/worker-affinity#specifying-worker-labels"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Learn more
                </a>
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {worker.labels.map(({ key, value }) => (
                  <Badge key={key} variant="outline">
                    {key}: {value}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Registered Workflows */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Registered Workflows</CardTitle>
            <CardDescription>
              Workflows that this worker can execute
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {filteredWorkflows.map((workflow) => (
                <Link
                  to={appRoutes.tenantWorkflowRoute.to}
                  params={{ tenant: tenantId, workflow: workflow.id }}
                  key={workflow.id}
                >
                  <Button variant="outline" size="sm">
                    {workflow.name}
                  </Button>
                </Link>
              ))}
            </div>
            {!showAllActions &&
              registeredWorkflows.length > N_ACTIONS_TO_PREVIEW && (
                <div className="mt-4 flex justify-center">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowAllActions(true)}
                  >
                    Show {registeredWorkflows.length - N_ACTIONS_TO_PREVIEW}{' '}
                    more
                  </Button>
                </div>
              )}
          </CardContent>
        </Card>

        {/* Recent Task Runs */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Recent Task Runs</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
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
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
