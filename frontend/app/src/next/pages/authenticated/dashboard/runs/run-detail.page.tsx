import { useParams } from 'react-router-dom';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { AlertCircle } from 'lucide-react';
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import { Skeleton } from '@/next/components/ui/skeleton';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { useEffect, useMemo, useCallback, useState } from 'react';
import { RunDataCard } from '@/next/components/runs/run-output-card';
import useTenant from '@/next/hooks/use-tenant';
import { WrongTenant } from '@/next/components/errors/unauthorized';
import { getFriendlyWorkflowRunId, RunId } from '@/next/components/runs/run-id';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { RunChildrenCardRoot } from '@/next/components/runs/run-children';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { MdOutlineReplay } from 'react-icons/md';
import { MdOutlineCancel } from 'react-icons/md';
import WorkflowRunVisualizer from '@/next/components/runs/run-dag/dag-run-visualizer';
import { SplitButton } from '@/next/components/ui/split-button';
import BasicLayout from '@/next/components/layouts/basic.layout';
import {
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Headline } from '@/next/components/ui/page-header';
import { TooltipContent } from '@/next/components/ui/tooltip';
import { TooltipTrigger } from '@/next/components/ui/tooltip';
import { Tooltip } from '@/next/components/ui/tooltip';
import { TooltipProvider } from '@/next/components/ui/tooltip';
import { Time } from '@/next/components/ui/time';
import { Duration } from '@/next/components/ui/duration';
import { V1TaskStatus } from '@/next/lib/api/generated/data-contracts';
import { ROUTES } from '@/next/lib/routes';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';

export default function RunDetailPage() {
  const { workflowRunId, taskId } = useParams<{
    workflowRunId: string;
    taskId: string;
  }>();
  const { tenant } = useTenant();
  const { data, isLoading, error, cancel, replay } = useRunDetail(
    workflowRunId || '',
  );
  const { data: parentData } = useRunDetail(
    data?.run?.parentTaskExternalId || '',
  );
  const [showTriggerModal, setShowTriggerModal] = useState(false);

  const { setBreadcrumbs } = useBreadcrumbs();

  const workflow = useMemo(() => data?.run, [data]);
  const tasks = useMemo(() => data?.tasks, [data]);
  const logs = useMemo(() => data?.taskEvents, [data]);

  const task = useMemo(() => {
    if (taskId) {
      return tasks?.find((t) => t.taskExternalId === taskId);
    }
    return tasks?.[0];
  }, [tasks, taskId]);

  useEffect(() => {
    if (!workflow) {
      return;
    }

    const breadcrumbs = [];

    if (parentData && parentData?.run) {
      const parentUrl = ROUTES.runs.parent(workflow);
      if (parentUrl) {
        breadcrumbs.push({
          title: getFriendlyWorkflowRunId(parentData.run) || '',
          label: <RunId wfRun={parentData.run} />,
          url: parentUrl,
          icon: () => <RunsBadge status={workflow?.status} variant="xs" />,
          alwaysShowIcon: true,
        });
      }
    }

    breadcrumbs.push({
      title: getFriendlyWorkflowRunId(workflow) || '',
      label: <RunId wfRun={workflow} />,
      url:
        task?.metadata.id === workflow?.metadata.id
          ? ROUTES.runs.detail(workflow.metadata.id)
          : ROUTES.runs.taskDetail(workflow.metadata.id, task!.taskExternalId),
      icon: () => <RunsBadge status={workflow?.status} variant="xs" />,
      alwaysShowIcon: true,
    });

    setBreadcrumbs(breadcrumbs);

    // Clear breadcrumbs when this component unmounts
    return () => {
      setBreadcrumbs([]);
    };
  }, [
    data?.tasks,
    parentData,
    workflow,
    workflowRunId,
    setBreadcrumbs,
    data?.run,
    task,
  ]);

  const canCancel = useMemo(() => {
    return (
      tasks &&
      tasks.length > 0 &&
      tasks.some(
        (t) =>
          t.status === V1TaskStatus.RUNNING || t.status === V1TaskStatus.QUEUED,
      )
    );
  }, [tasks]);

  const canCancelRunning = useMemo(() => {
    return tasks && tasks.some((t) => t.status === V1TaskStatus.RUNNING);
  }, [tasks]);

  const canCancelQueued = useMemo(() => {
    return tasks && tasks.some((t) => t.status === V1TaskStatus.QUEUED);
  }, [tasks]);

  const cancelRunningTasks = useMemo(() => {
    return tasks?.filter((t) => t.status === V1TaskStatus.RUNNING) || [];
  }, [tasks]);

  const cancelQueuedTasks = useMemo(() => {
    return tasks?.filter((t) => t.status === V1TaskStatus.QUEUED) || [];
  }, [tasks]);

  const canReplay = useMemo(() => {
    return tasks && tasks.length > 0;
  }, [tasks]);

  const getTasksByStatus = useCallback(
    (status: V1TaskStatus) => {
      return tasks?.filter((t) => t.status === status) || [];
    },
    [tasks],
  );

  const canReplayFailed = useMemo(
    () => getTasksByStatus(V1TaskStatus.FAILED).length > 0,
    [getTasksByStatus],
  );

  const canReplayCompleted = useMemo(
    () => getTasksByStatus(V1TaskStatus.COMPLETED).length > 0,
    [getTasksByStatus],
  );

  const canReplayCanceled = useMemo(
    () => getTasksByStatus(V1TaskStatus.CANCELLED).length > 0,
    [getTasksByStatus],
  );

  const canReplayRunning = useMemo(
    () => getTasksByStatus(V1TaskStatus.RUNNING).length > 0,
    [getTasksByStatus],
  );

  const replayFailedTasks = useMemo(
    () => getTasksByStatus(V1TaskStatus.FAILED),
    [getTasksByStatus],
  );
  const replayCompletedTasks = useMemo(
    () => getTasksByStatus(V1TaskStatus.COMPLETED),
    [getTasksByStatus],
  );
  const replayCanceledTasks = useMemo(
    () => getTasksByStatus(V1TaskStatus.CANCELLED),
    [getTasksByStatus],
  );
  const replayRunningTasks = useMemo(
    () => getTasksByStatus(V1TaskStatus.RUNNING),
    [getTasksByStatus],
  );

  if (isLoading) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4">
        <div className="rounded-lg bg-card p-4">
          <div className="flex flex-col gap-4">
            <Skeleton className="h-10 w-3/4" />
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Skeleton className="h-32" />
              <Skeleton className="h-32" />
            </div>
            <Skeleton className="h-64" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !workflow) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error loading run</AlertTitle>
          <AlertDescription>
            {error instanceof Error
              ? error.message
              : 'Failed to load run details'}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // wrong tenant selected error
  if (tenant?.metadata.id !== workflow.tenantId) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4">
        {workflow?.tenantId && (
          <WrongTenant desiredTenantId={workflow.tenantId} />
        )}
      </div>
    );
  }

  const Timing = () => {
    return (
      <div className="flex flex-col items-end sm:flex-row sm:items-center sm:justify-start gap-x-4 gap-y-2 text-sm text-muted-foreground">
        <div className="flex items-center gap-2">
          <span>Created</span>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="whitespace-nowrap">
                  <Time date={workflow.createdAt} variant="timestamp" />
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <Time date={workflow.createdAt} variant="timeSince" />
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>

        <div className="flex items-center gap-2">
          <span>Started</span>
          {workflow.startedAt ? (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="whitespace-nowrap">
                    <Time date={workflow.startedAt} variant="timeSince" />
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  <Time date={workflow.startedAt} variant="timestamp" />
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            <span>Not started</span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <span>Duration</span>
          <span className="whitespace-nowrap">
            <Duration
              start={workflow.startedAt}
              end={workflow.finishedAt}
              status={workflow.status}
            />
          </span>
        </div>
      </div>
    );
  };

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description={<Timing />}>
          <h1 className="text-2xl font-bold truncate flex items-center gap-2">
            <RunsBadge status={workflow.status} variant="xs" />
            <RunId wfRun={workflow} />
          </h1>
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <SplitButton
              tooltip={
                canCancel
                  ? 'Cancel the run'
                  : 'Cannot cancel the run because it is not running or queued'
              }
              variant="outline"
              size="icon"
              disabled={!canCancel || cancel.isPending}
              onClick={async () => cancel.mutateAsync({ tasks: tasks || [] })}
              dropdownItems={[
                {
                  label: 'Cancel Running',
                  onClick: async () =>
                    cancel.mutateAsync({ tasks: cancelRunningTasks }),
                  disabled: !canCancelRunning || cancel.isPending,
                },
                {
                  label: 'Cancel Queued',
                  onClick: async () =>
                    cancel.mutateAsync({ tasks: cancelQueuedTasks }),
                  disabled: !canCancelQueued || cancel.isPending,
                },
              ]}
            >
              <MdOutlineCancel className="h-4 w-4" />
              <span className="sr-only">Cancel</span>
            </SplitButton>
          </HeadlineActionItem>
          <HeadlineActionItem>
            <SplitButton
              tooltip={
                canReplay
                  ? 'Replay the run'
                  : 'Cannot replay the run because there are no tasks'
              }
              variant="outline"
              size="icon"
              disabled={!canReplay || replay.isPending}
              onClick={async () => replay.mutateAsync({ tasks: tasks || [] })}
              dropdownItems={[
                {
                  label: 'Replay As New Run',
                  onClick: () => setShowTriggerModal(true),
                },
                {
                  label: 'Replay Failed',
                  onClick: async () =>
                    replay.mutateAsync({ tasks: replayFailedTasks }),
                  disabled: !canReplayFailed || replay.isPending,
                },
                {
                  label: 'Replay Completed',
                  onClick: async () =>
                    replay.mutateAsync({ tasks: replayCompletedTasks }),
                  disabled: !canReplayCompleted || replay.isPending,
                },
                {
                  label: 'Replay Canceled',
                  onClick: async () =>
                    replay.mutateAsync({ tasks: replayCanceledTasks }),
                  disabled: !canReplayCanceled || replay.isPending,
                },
                {
                  label: 'Replay Running',
                  onClick: async () =>
                    replay.mutateAsync({ tasks: replayRunningTasks }),
                  disabled: !canReplayRunning || replay.isPending,
                },
              ]}
            >
              <MdOutlineReplay className="h-4 w-4" />
              <span className="sr-only">Replay</span>
            </SplitButton>
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      {workflowRunId && (
        <div className="w-full overflow-x-auto">
          <WorkflowRunVisualizer workflowRunId={workflowRunId} />
        </div>
      )}

      <div className="grid grid-cols-1 gap-4">
        <RunChildrenCardRoot workflow={workflow} parentRun={parentData?.run} />
      </div>

      <div className="grid grid-cols-1 gap-4">
        <RunDataCard
          title="Logs"
          output={logs}
          status={workflow.status}
          variant="output"
          collapsed
        />

        <RunDataCard
          title="Input"
          output={(workflow.input as { input: object }).input}
          status={workflow.status}
          variant="input"
        />
        {tasks
          ?.sort(
            (a, b) =>
              new Date(b.startedAt || '').getTime() -
              new Date(a.startedAt || '').getTime(),
          )
          .map((task) => (
            <RunDataCard
              key={task.displayName}
              title={task.displayName}
              description={
                task.errorMessage
                  ? 'The task failed to complete with the following error'
                  : 'The task completed successfully with output'
              }
              output={task.output}
              error={task.errorMessage}
              status={task.status}
              variant="output"
            />
          ))}
        <RunDataCard
          title="Metadata"
          output={{
            taskRunId: task?.metadata.id,
            workflowRunId: workflow.metadata.id,
            additional: workflow.additionalMetadata,
          }}
          status={workflow.status}
          variant="metadata"
          collapsed
          actions={
            <div className="flex items-center gap-2">
              <DocsButton doc={docs.home['additional-metadata']} size="icon" />
            </div>
          }
        />
        <RunDataCard
          title="Configuration"
          output={{
            todo: 'TODO',
          }}
          status={workflow.status}
          variant="metadata"
          collapsed
        />
      </div>

      <TriggerRunModal
        show={showTriggerModal}
        onClose={() => setShowTriggerModal(false)}
        defaultInput={JSON.stringify((workflow.input as any).input, null, 2)}
        defaultAddlMeta={JSON.stringify(workflow.additionalMetadata, null, 2)}
        defaultWorkflowId={workflow.workflowId}
      />
    </BasicLayout>
  );
}
