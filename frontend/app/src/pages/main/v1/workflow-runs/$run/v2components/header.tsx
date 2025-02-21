import React from 'react';
import { Link, useOutletContext } from 'react-router-dom';

import CronPrettifier from 'cronstrue';

import api, {
  V2TaskStatus,
  WorkflowRunShape,
  WorkflowRunStatus,
  queries,
} from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { useApiError } from '@/lib/hooks';
import { TenantContextType } from '@/lib/outlet';
import { Button } from '@/components/v1/ui/button';
import {
  AdjustmentsHorizontalIcon,
  ArrowPathIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { ArrowTopRightIcon } from '@radix-ui/react-icons';
import {
  Breadcrumb,
  BreadcrumbList,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbSeparator,
  BreadcrumbPage,
} from '@/components/v1/ui/breadcrumb';
import { formatDuration } from '@/lib/utils';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { useToast } from '@/components/v1/hooks/use-toast';
import { useTenant } from '@/lib/atoms';

interface RunDetailHeaderProps {
  data?: WorkflowRunShape;
  loading?: boolean;
  refetch: () => void;
}

interface V2RunDetailHeaderProps {
  taskRunId: string;
  loading?: boolean;
}

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

const RunDetailHeader: React.FC<RunDetailHeaderProps> = ({
  data,
  loading,
  refetch,
}) => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const { toast } = useToast();

  const { handleApiError } = useApiError({});

  const cancelWorkflowRunMutation = useMutation({
    mutationKey: ['workflow-run:cancel', data?.tenantId, data?.metadata.id],
    onMutate: () => {
      toast({
        title: 'Cancelling workflow run...',
        duration: 3000,
      });
    },
    mutationFn: async () => {
      const tenantId = data?.tenantId;
      const workflowRunId = data?.metadata.id;

      invariant(tenantId, 'has tenantId');
      invariant(workflowRunId, 'has tenantId');

      const res = await api.workflowRunCancel(tenantId, {
        workflowRunIds: [workflowRunId],
      });

      return res.data;
    },
    onError: handleApiError,
  });

  const replayWorkflowRunsMutation = useMutation({
    mutationKey: ['workflow-run:update:replay', tenant.metadata.id],
    onMutate: () => {
      toast({
        title: 'Replaying workflow run...',
        duration: 3000,
      });
    },
    mutationFn: async () => {
      if (!data) {
        return;
      }

      await api.workflowRunUpdateReplay(tenant.metadata.id, {
        workflowRunIds: [data?.metadata.id],
      });
    },
    onSuccess: () => {
      refetch();
    },
    onError: handleApiError,
  });

  if (loading || !data) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col gap-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href="/">Home</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink href="/workflow-runs">Workflow Runs</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{data.displayName}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div>
            <h2 className="text-2xl font-bold leading-tight text-foreground flex flex-row gap-4 items-center">
              <AdjustmentsHorizontalIcon className="w-5 h-5 mt-1" />
              {data?.displayName}
            </h2>
          </div>
          <div className="flex flex-row gap-2 items-center">
            <a
              href={`/workflows/${data.workflowId}`}
              target="_blank"
              rel="noreferrer"
            >
              <Button size={'sm'} className="px-2 py-2 gap-2" variant="outline">
                <ArrowTopRightIcon className="w-4 h-4" />
                Workflow Definition
              </Button>
            </a>
            <Button
              size={'sm'}
              className="px-2 py-2 gap-2"
              variant={'outline'}
              disabled={!WORKFLOW_RUN_TERMINAL_STATUSES.includes(data.status)}
              onClick={() => {
                replayWorkflowRunsMutation.mutate();
              }}
            >
              <ArrowPathIcon className="w-4 h-4" />
              Replay
            </Button>
            <Button
              size={'sm'}
              className="px-2 py-2 gap-2"
              variant={'outline'}
              disabled={WORKFLOW_RUN_TERMINAL_STATUSES.includes(data.status)}
              onClick={() => {
                cancelWorkflowRunMutation.mutate();
              }}
            >
              <XCircleIcon className="w-4 h-4" />
              Cancel
            </Button>
          </div>
        </div>
      </div>
      {data.triggeredBy?.parentWorkflowRunId && (
        <TriggeringParentWorkflowRunSection
          tenantId={data.tenantId}
          parentWorkflowRunId={data.triggeredBy.parentWorkflowRunId}
        />
      )}
      {data.triggeredBy?.eventId && (
        <TriggeringEventSection eventId={data.triggeredBy.eventId} />
      )}
      {data.triggeredBy?.cronSchedule && (
        <TriggeringCronSection cron={data.triggeredBy.cronSchedule} />
      )}
      <div className="flex flex-row gap-2 items-center">
        <RunSummary data={data} />
      </div>
    </div>
  );
};

export const V2RunDetailHeader: React.FC<V2RunDetailHeaderProps> = ({
  taskRunId,
}) => {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const { isLoading: loading, data } = useQuery({
    ...queries.v2Tasks.get(taskRunId),
  });

  if (loading || !data) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col gap-4">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href="/">Home</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink href="/workflow-runs">Workflow Runs</BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{data.displayName}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div>
            <h2 className="text-2xl font-bold leading-tight text-foreground flex flex-row gap-4 items-center">
              <AdjustmentsHorizontalIcon className="w-5 h-5 mt-1" />
              {data?.displayName}
            </h2>
          </div>
          <div className="flex flex-row gap-2 items-center">
            <a
              href={`/workflows/${data.workflowId}`}
              target="_blank"
              rel="noreferrer"
            >
              <Button size={'sm'} className="px-2 py-2 gap-2" variant="outline">
                <ArrowTopRightIcon className="w-4 h-4" />
                Workflow Definition
              </Button>
            </a>
            <Button
              size={'sm'}
              className="px-2 py-2 gap-2"
              variant={'outline'}
              disabled
            >
              <ArrowPathIcon className="w-4 h-4" />
              Replay
            </Button>
            <Button
              size={'sm'}
              className="px-2 py-2 gap-2"
              variant={'outline'}
              disabled
            >
              <XCircleIcon className="w-4 h-4" />
              Cancel
            </Button>
          </div>
        </div>
      </div>
      {/* {data.triggeredBy?.parentWorkflowRunId && (
        <TriggeringParentWorkflowRunSection
          tenantId={data.tenantId}
          parentWorkflowRunId={data.triggeredBy.parentWorkflowRunId}
        />
      )} */}
      {/* {data.triggeredBy?.eventId && (
        <TriggeringEventSection eventId={data.triggeredBy.eventId} />
      )} */}
      {/* {data.triggeredBy?.cronSchedule && (
        <TriggeringCronSection cron={data.triggeredBy.cronSchedule} />
      )} */}
      <div className="flex flex-row gap-2 items-center">
        <V2RunSummary taskRunId={taskRunId} />
      </div>
    </div>
  );
};

export default RunDetailHeader;

const RunSummary: React.FC<{ data: WorkflowRunShape }> = ({ data }) => {
  const timings = [];

  timings.push(
    <div key="created" className="text-sm text-muted-foreground">
      {'Created '}
      <RelativeDate date={data.metadata.createdAt} />
    </div>,
  );

  if (data.startedAt) {
    timings.push(
      <div key="created" className="text-sm text-muted-foreground">
        {'Started '}
        <RelativeDate date={data.startedAt} />
      </div>,
    );
  } else {
    timings.push(
      <div key="created" className="text-sm text-muted-foreground">
        Running
      </div>,
    );
  }

  if (data.status === WorkflowRunStatus.CANCELLED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Cancelled '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === WorkflowRunStatus.FAILED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Failed '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === WorkflowRunStatus.SUCCEEDED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Succeeded '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.duration) {
    timings.push(
      <div key="duration" className="text-sm text-muted-foreground">
        Run took {formatDuration(data.duration)}
      </div>,
    );
  }

  // interleave the timings with a dot
  const interleavedTimings: JSX.Element[] = [];

  timings.forEach((timing, index) => {
    interleavedTimings.push(timing);
    if (index < timings.length - 1) {
      interleavedTimings.push(
        <div key={`dot-${index}`} className="text-sm text-muted-foreground">
          |
        </div>,
      );
    }
  });

  return (
    <div className="flex flex-row gap-4 items-center">{interleavedTimings}</div>
  );
};

export const V2RunSummary = ({ taskRunId }: { taskRunId: string }) => {
  const { tenant } = useTenant();
  invariant(tenant);

  const { data } = useQuery({
    ...queries.v2Tasks.get(taskRunId),
  });

  const timings = [];

  if (!data) {
    return null;
  }

  timings.push(
    <div key="created" className="text-sm text-muted-foreground">
      {'Created '}
      <RelativeDate date={data.metadata.createdAt} />
    </div>,
  );

  if (data.startedAt) {
    timings.push(
      <div key="started" className="text-sm text-muted-foreground">
        {'Started '}
        <RelativeDate date={data.startedAt} />
      </div>,
    );
  } else {
    timings.push(
      <div key="running" className="text-sm text-muted-foreground">
        Running
      </div>,
    );
  }

  if (data.status === V2TaskStatus.CANCELLED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Cancelled '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === V2TaskStatus.FAILED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Failed '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === V2TaskStatus.COMPLETED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Succeeded '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.duration) {
    timings.push(
      <div key="duration" className="text-sm text-muted-foreground">
        Run took {formatDuration(data.duration)}
      </div>,
    );
  }

  // interleave the timings with a dot
  const interleavedTimings: JSX.Element[] = [];

  timings.forEach((timing, index) => {
    interleavedTimings.push(timing);
    if (index < timings.length - 1) {
      interleavedTimings.push(
        <div key={`dot-${index}`} className="text-sm text-muted-foreground">
          |
        </div>,
      );
    }
  });

  return (
    <div className="flex flex-row gap-4 items-center">{interleavedTimings}</div>
  );
};

function TriggeringParentWorkflowRunSection({
  tenantId,
  parentWorkflowRunId,
}: {
  tenantId: string;
  parentWorkflowRunId: string;
}) {
  // get the parent workflow run id
  const workflowRunQuery = useQuery({
    ...queries.workflowRuns.get(tenantId, parentWorkflowRunId),
  });

  if (workflowRunQuery.isLoading || !workflowRunQuery.data) {
    return null;
  }

  const workflowRun = workflowRunQuery.data;

  return (
    <div className="text-sm text-gray-700 dark:text-gray-300 flex flex-row gap-1">
      Triggered by
      <Link
        to={`/workflow-runs/${parentWorkflowRunId}`}
        className="font-semibold hover:underline  text-indigo-500 dark:text-indigo-200"
      >
        {workflowRun.displayName} ➶
      </Link>
    </div>
  );
}

function TriggeringEventSection({ eventId }: { eventId: string }) {
  // get the parent workflow run id
  const eventData = useQuery({
    ...queries.events.get(eventId),
  });

  if (eventData.isLoading || !eventData.data) {
    return null;
  }

  const event = eventData.data;

  return (
    <div className="text-sm text-gray-700 dark:text-gray-300 flex flex-row gap-1">
      Triggered by {event.key}
    </div>
  );
}

function TriggeringCronSection({ cron }: { cron: string }) {
  const prettyInterval = `runs ${CronPrettifier.toString(
    cron,
  ).toLowerCase()} UTC`;

  return (
    <div className="text-sm text-gray-700 dark:text-gray-300">
      Triggered by cron {cron} which {prettyInterval}
    </div>
  );
}
