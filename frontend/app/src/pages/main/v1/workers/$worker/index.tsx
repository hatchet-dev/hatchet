import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { flattenDAGsKey } from '../../workflow-runs-v1/components/v1/task-runs-columns';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';
import { DocsButton } from '@/components/v1/docs/docs-button';
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
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, { queries, UpdateWorkerRequest, Worker } from '@/lib/api';
import { shouldRetryQueryError } from '@/lib/error-utils';
import { docsPages } from '@/lib/generated/docs';
import { useApiError } from '@/lib/hooks';
import { capitalize, cn } from '@/lib/utils';
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

export default function WorkerDetail() {
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
  const usedSlots = maxSlots - availableSlots;
  const usedPercentage =
    maxSlots > 0 ? Math.round((usedSlots / maxSlots) * 100) : 0;
  const availableDurableSlots = worker.durableAvailableRuns ?? 0;
  const maxDurableSlots = worker.durableMaxRuns ?? 0;
  const usedDurableSlots = maxDurableSlots - availableDurableSlots;
  const usedDurablePercentage =
    maxDurableSlots > 0
      ? Math.round((usedDurableSlots / maxDurableSlots) * 100)
      : 0;

  // dynamically set the max columns in the grid based on the presence of runtime info and labels
  const maxCols =
    2 +
    Number(!!worker.runtimeInfo) +
    Number((worker?.labels?.length ?? 0) > 0);

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto p-4">
        <div className="flex flex-row items-center justify-between">
          <div className="flex flex-row items-center gap-4">
            <ServerStackIcon className="mt-1 h-6 w-6 text-foreground" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {worker.name}
            </h2>
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

        <div
          className={cn(
            'mt-6 grid gap-4 md:grid-cols-2',
            `lg:grid-cols-${maxCols}`,
          )}
        >
          <Card
            variant="light"
            className="h-52 overflow-y-auto bg-background border-none"
          >
            <CardHeader>
              <CardTitle>Connection Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div>
                <div className="text-gray-500 dark:text-gray-400">
                  First Connected
                </div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  <RelativeDate date={worker.metadata?.createdAt} />
                </div>
              </div>
              {worker.lastListenerEstablished && (
                <div>
                  <div className="text-gray-500 dark:text-gray-400">
                    Last Listener Established
                  </div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">
                    <RelativeDate date={worker.lastListenerEstablished} />
                  </div>
                </div>
              )}
              <div>
                <div className="text-gray-500 dark:text-gray-400">
                  Last Heartbeat
                </div>
                <div className="font-medium text-gray-900 dark:text-gray-100">
                  {worker.lastHeartbeatAt ? (
                    <RelativeDate date={worker.lastHeartbeatAt} />
                  ) : (
                    'Never'
                  )}
                </div>
              </div>
            </CardContent>
          </Card>

          <Card
            variant="light"
            className="h-52 overflow-y-auto bg-background border-none"
          >
            <CardHeader>
              <CardTitle>Available Run Slots</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="space-y-2">
                <div className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  Standard
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="text-3xl font-bold text-gray-900 dark:text-gray-100">
                    {maxSlots > 0 ? availableSlots : '∞'}
                  </span>
                  {maxSlots > 0 && (
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      / {maxSlots} total
                    </span>
                  )}
                </div>
                {maxSlots > 0 && (
                  <div className="space-y-1">
                    <div className="h-2 w-full overflow-hidden rounded-full bg-gray-600/40 dark:bg-gray-500/50 ">
                      <div
                        className="h-full bg-emerald-300 dark:bg-emerald-500 transition-all"
                        style={{ width: `${usedPercentage}%` }}
                      />
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      {usedSlots} used, {availableSlots} available
                    </div>
                  </div>
                )}
              </div>
              {maxDurableSlots > 0 && (
                <div className="space-y-2">
                  <div className="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
                    Durable
                  </div>
                  <div className="flex items-baseline gap-2">
                    <span className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
                      {availableDurableSlots}
                    </span>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      / {maxDurableSlots} total
                    </span>
                  </div>
                  <div className="space-y-1">
                    <div className="h-2 w-full overflow-hidden rounded-full bg-gray-600/40 dark:bg-gray-500/50 ">
                      <div
                        className="h-full bg-sky-300 dark:bg-sky-500 transition-all"
                        style={{ width: `${usedDurablePercentage}%` }}
                      />
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400">
                      {usedDurableSlots} used, {availableDurableSlots} available
                    </div>
                  </div>
                </div>
              )}
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Slots represent concurrent task runs.{' '}
                <DocsButton
                  variant="text"
                  doc={docsPages.home.workers}
                  label="Learn more"
                  scrollTo={'understanding-slots'}
                />
              </p>
            </CardContent>
          </Card>

          {worker.runtimeInfo &&
            (worker.runtimeInfo?.sdkVersion ||
              worker.runtimeInfo?.languageVersion ||
              worker.runtimeInfo?.os) && (
              <Card
                variant="light"
                className="h-52 overflow-y-auto bg-background border-none"
              >
                <CardHeader>
                  <CardTitle>Runtime Info</CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm">
                  {worker.runtimeInfo?.os && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400">OS</div>
                      <div className="font-medium text-gray-900 dark:text-gray-100">
                        {worker.runtimeInfo.os}
                      </div>
                    </div>
                  )}
                  {worker.runtimeInfo?.languageVersion && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400">
                        Runtime
                      </div>
                      <div className="font-medium text-gray-900 dark:text-gray-100">
                        {capitalize(worker.runtimeInfo.language ?? '')}{' '}
                        {worker.runtimeInfo.languageVersion}
                      </div>
                    </div>
                  )}
                  {worker.runtimeInfo?.sdkVersion && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400">
                        Hatchet SDK
                      </div>
                      <div className="font-medium text-gray-900 dark:text-gray-100">
                        {worker.runtimeInfo.sdkVersion}
                      </div>
                    </div>
                  )}
                  {worker.runtimeInfo?.runtimeExtra && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400">
                        Runtime Extra
                      </div>
                      <div className="font-medium text-gray-900 dark:text-gray-100">
                        {worker.runtimeInfo.runtimeExtra}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

          {worker.labels && worker.labels.length > 0 && (
            <Card
              variant="light"
              className="h-52 overflow-y-auto bg-background border-none"
            >
              <CardHeader>
                <CardTitle>Worker Labels</CardTitle>
                <CardDescription>
                  Key-value pairs used to prioritize step assignment to specific
                  workers.{' '}
                  <DocsButton
                    variant="text"
                    doc={docsPages.home['worker-affinity']}
                    label="Learn more"
                    scrollTo={'specifying-worker-labels'}
                  />
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {worker.labels.map(({ key, value }) => (
                    <Badge key={key} variant="secondary">
                      {key}: {value}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </div>

        <Card
          variant="light"
          className="mt-4 overflow-y-auto bg-background border-none"
        >
          <CardHeader>
            <CardTitle>Registered Workflows</CardTitle>
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
                <div className="mt-3 flex justify-start">
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

        <Card variant="light" className="mt-4 bg-primary border-none">
          <CardContent className="flex-1 h-96 overflow-y-auto bg-background">
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
              <RunsTable leftLabel={'Recent runs'} />
            </RunsProvider>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
