import { V1TaskStatus, WorkflowRunStatus } from '@/lib/api';
import { AdjustmentsHorizontalIcon } from '@heroicons/react/24/outline';
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
import { useWorkflowDetails } from '../../hooks/workflow-details';
import { TaskRunActionButton } from '../../../task-runs-v1/actions';
import { TASK_RUN_TERMINAL_STATUSES } from './step-run-detail/step-run-detail';
import { WorkflowDefinitionLink } from '@/pages/main/workflow-runs/$run/v2components/workflow-definition';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export const WORKFLOW_RUN_TERMINAL_STATUSES = [
  WorkflowRunStatus.CANCELLED,
  WorkflowRunStatus.FAILED,
  WorkflowRunStatus.SUCCEEDED,
];

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
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href={`/tenants/${tenantId}/runs`}>
              Home
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink href={`/tenants/${tenantId}/runs`}>
              Runs
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{workflowRun.displayName}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div>
            <h2 className="text-2xl font-bold leading-tight text-foreground flex flex-row gap-4 items-center">
              <AdjustmentsHorizontalIcon className="w-5 h-5 mt-1" />
              {workflowRun.displayName}
            </h2>
          </div>
          <div className="flex flex-row gap-2 items-center">
            <CopyWorkflowConfigButton workflowConfig={workflowConfig} />
            <WorkflowDefinitionLink workflowId={workflowRun.workflowId} />
            <TaskRunActionButton
              actionType="replay"
              params={{ externalIds: [workflowRun.metadata.id] }}
              disabled={
                !TASK_RUN_TERMINAL_STATUSES.includes(workflowRun.status)
              }
              showModal={false}
            />
            <TaskRunActionButton
              actionType="cancel"
              params={{ externalIds: [workflowRun.metadata.id] }}
              disabled={TASK_RUN_TERMINAL_STATUSES.includes(workflowRun.status)}
              showModal={false}
            />
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
        <V1RunSummary />
      </div>
    </div>
  );
};

export const V1RunSummary = () => {
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
    <div className="flex flex-row gap-4 items-center">{interleavedTimings}</div>
  );
};
