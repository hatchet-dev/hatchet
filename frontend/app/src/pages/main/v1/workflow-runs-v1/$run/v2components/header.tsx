import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { useWorkflowDetails } from '../../hooks/use-workflow-details';
import { TASK_RUN_TERMINAL_STATUSES } from './step-run-detail/step-run-detail';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { Toaster } from '@/components/v1/ui/toaster';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { V1TaskStatus, queries } from '@/lib/api';
import { formatDuration } from '@/lib/utils';
import { WorkflowDefinitionLink } from '@/pages/main/workflow-runs/$run/v2components/workflow-definition';
import { appRoutes } from '@/router';
import { AdjustmentsHorizontalIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';

export const V1RunDetailHeader = () => {
  const { tenantId } = useCurrentTenantId();
  const {
    workflowRun,
    workflowConfig,
    isLoading: loading,
  } = useWorkflowDetails();

  if (loading || !workflowRun) {
    return <div>Loading...</div>;
  }

  return (
    <div className="flex flex-col gap-4">
      <Toaster />
      <div className="flex flex-row items-center justify-between">
        <div className="flex w-full flex-row items-center justify-between">
          <div>
            <h2 className="flex flex-row items-center gap-4 text-2xl font-bold leading-tight text-foreground">
              <AdjustmentsHorizontalIcon className="mt-1 h-5 w-5" />
              {workflowRun.displayName}
            </h2>
          </div>
          <div className="flex flex-row items-center gap-2">
            <CopyWorkflowConfigButton workflowConfig={workflowConfig} />
            <WorkflowDefinitionLink workflowId={workflowRun.workflowId} />
            <TaskRunActionButton
              actionType="replay"
              paramOverrides={{ externalIds: [workflowRun.metadata.id] }}
              disabled={
                !TASK_RUN_TERMINAL_STATUSES.includes(workflowRun.status)
              }
              showModal={false}
              showLabel
            />
            <TaskRunActionButton
              actionType="cancel"
              paramOverrides={{ externalIds: [workflowRun.metadata.id] }}
              disabled={TASK_RUN_TERMINAL_STATUSES.includes(workflowRun.status)}
              showModal={false}
              showLabel
            />
          </div>
        </div>
      </div>
      {workflowRun.parentTaskExternalId && (
        <TriggeringParentWorkflowRunSection
          tenantId={tenantId}
          parentTaskExternalId={workflowRun.parentTaskExternalId}
        />
      )}
      {/* {data.triggeredBy?.eventId && (
        <TriggeringEventSection eventId={data.triggeredBy.eventId} />
      )} */}
      {/* {data.triggeredBy?.cronSchedule && (
        <TriggeringCronSection cron={data.triggeredBy.cronSchedule} />
      )} */}
      <div className="flex flex-row items-center gap-2">
        <V1RunSummary />
      </div>
    </div>
  );
};

const V1RunSummary = () => {
  const { workflowRun } = useWorkflowDetails();

  const timings = [];

  if (!workflowRun) {
    return null;
  }

  timings.push(
    <div key="created" className="text-sm text-muted-foreground">
      {'Created '}
      <RelativeDate date={workflowRun.createdAt} />
    </div>,
  );

  if (workflowRun.startedAt) {
    timings.push(
      <div key="started" className="text-sm text-muted-foreground">
        {'Started '}
        <RelativeDate date={workflowRun.startedAt} />
      </div>,
    );
  } else {
    timings.push(
      <div key="running" className="text-sm text-muted-foreground">
        Running
      </div>,
    );
  }

  if (workflowRun.status === V1TaskStatus.CANCELLED && workflowRun.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Cancelled '}
        <RelativeDate date={workflowRun.finishedAt} />
      </div>,
    );
  }

  if (workflowRun.status === V1TaskStatus.FAILED && workflowRun.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Failed '}
        <RelativeDate date={workflowRun.finishedAt} />
      </div>,
    );
  }

  if (workflowRun.status === V1TaskStatus.COMPLETED && workflowRun.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Succeeded '}
        <RelativeDate date={workflowRun.finishedAt} />
      </div>,
    );
  }

  if (workflowRun.duration) {
    timings.push(
      <div key="duration" className="text-sm text-muted-foreground">
        Run took {formatDuration(workflowRun.duration)}
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
    <div className="flex flex-row items-center gap-4">{interleavedTimings}</div>
  );
};

function TriggeringParentWorkflowRunSection({
  tenantId,
  parentTaskExternalId,
}: {
  tenantId: string;
  parentTaskExternalId: string;
}) {
  // Get the parent task to find the parent workflow run
  const parentTaskQuery = useQuery({
    ...queries.v1Tasks.get(parentTaskExternalId),
  });

  const parentTask = parentTaskQuery.data;
  const parentWorkflowRunId = parentTask?.workflowRunExternalId;

  // Get the parent workflow run details - only enabled when we have a parent workflow run ID
  const parentWorkflowRunQuery = useQuery({
    ...queries.v1WorkflowRuns.details(parentWorkflowRunId || ''),
    enabled: !!parentWorkflowRunId,
  });

  // Show nothing while loading or if no data
  if (parentTaskQuery.isLoading || !parentTask || !parentWorkflowRunId) {
    return null;
  }

  if (parentWorkflowRunQuery.isLoading || !parentWorkflowRunQuery.data) {
    return null;
  }

  const parentWorkflowRun = parentWorkflowRunQuery.data.run;

  return (
    <div className="flex flex-row gap-1 text-sm text-gray-700 dark:text-gray-300">
      Triggered by
      <Link
        to={appRoutes.tenantRunRoute.to}
        params={{ tenant: tenantId, run: parentWorkflowRunId }}
        className="font-semibold text-indigo-500 hover:underline dark:text-indigo-200"
      >
        {parentWorkflowRun.displayName} ➶
      </Link>
    </div>
  );
}
