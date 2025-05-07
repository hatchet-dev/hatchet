import { useParams, useNavigate } from 'react-router-dom';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { AlertCircle } from 'lucide-react';
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import { Skeleton } from '@/next/components/ui/skeleton';
import { useMemo, useCallback, useState } from 'react';
import useTenant from '@/next/hooks/use-tenant';
import { WrongTenant } from '@/next/components/errors/unauthorized';
import { getFriendlyWorkflowRunId, RunId } from '@/next/components/runs/run-id';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { MdOutlineReplay } from 'react-icons/md';
import { MdOutlineCancel } from 'react-icons/md';
import WorkflowRunVisualizer from '@/next/components/runs/run-dag/dag-run-visualizer';
import { SplitButton } from '@/next/components/ui/split-button';
import { SheetViewLayout } from '@/next/components/layouts/sheet-view.layout';
import {
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
} from '@/next/components/ui/page-header';
import { Headline } from '@/next/components/ui/page-header';
import { Duration } from '@/next/components/ui/duration';
import { V1TaskStatus } from '@/lib/api/generated/data-contracts';
import { ROUTES } from '@/next/lib/routes';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/components/v1/ui/tabs';
import { RunEventLog } from '@/next/components/runs/run-event-log/run-event-log';
import { FilterProvider } from '@/next/hooks/utils/use-filters';
import { RunDetailSheet } from './run-detail-sheet';
import { Separator } from '@/next/components/ui/separator';
import { Waterfall } from '@/next/components/waterfall/waterfall';
import RelativeDate from '@/next/components/ui/relative-date';
import { useBreadcrumbs } from '@/next/hooks/use-breadcrumbs';
import { WorkflowDetailsProvider } from '@/next/hooks/use-workflow-details';
import WorkflowGeneralSettings from '../workflows/settings';

export default function RunDetailPage() {
  const { workflowRunId, taskId } = useParams<{
    workflowRunId: string;
    taskId: string;
  }>();
  return (
    <RunDetailProvider runId={workflowRunId || ''} defaultRefetchInterval={1000}>
      <RunDetailPageContent workflowRunId={workflowRunId} taskId={taskId} />
    </RunDetailProvider>
  );
}

type RunDetailPageProps = {
  workflowRunId?: string;
  taskId?: string;
};

function RunDetailPageContent({ workflowRunId, taskId }: RunDetailPageProps) {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const { data, isLoading, error, cancel, replay, parentData, lastRefetchTime, refetchInterval } = useRunDetail();

  const [showTriggerModal, setShowTriggerModal] = useState(false);

  const workflow = useMemo(() => data?.run, [data]);
  const tasks = useMemo(() => data?.tasks, [data]);

  const selectedTask = useMemo(() => {
    if (taskId) {
      return tasks?.find((t) => t.taskExternalId === taskId);
    }
    return tasks?.[0];
  }, [tasks, taskId]);

  const handleTaskSelect = useCallback(
    (taskId: string) => {
      navigate(ROUTES.runs.taskDetail(workflowRunId!, taskId));
    },
    [navigate, workflowRunId],
  );

  const handleCloseSheet = useCallback(() => {
    navigate(ROUTES.runs.detail(workflowRunId!));
  }, [navigate, workflowRunId]);

  // useBreadcrumbs(() => {
  //   if (!workflow) {
  //     return [];
  //   }

  //   const breadcrumbs = [];

  //   if (parentData) {
  //     const parentUrl = ROUTES.runs.detail(parentData.run.metadata.id);
  //     breadcrumbs.push({
  //       title: getFriendlyWorkflowRunId(parentData.run) || '',
  //       label: <RunId wfRun={parentData.run} />,
  //       url: parentUrl,
  //       icon: () => <RunsBadge status={workflow?.status} variant="xs" />,
  //       alwaysShowIcon: true,
  //     });
  //   }

  //   breadcrumbs.push({
  //     title: getFriendlyWorkflowRunId(workflow) || '',
  //     label: <RunId wfRun={workflow} />,
  //     url:
  //       selectedTask?.metadata.id === workflow?.metadata.id
  //         ? ROUTES.runs.detail(workflow.metadata.id)
  //         : ROUTES.runs.taskDetail(
  //             workflow.metadata.id,
  //             selectedTask?.taskExternalId || '',
  //           ),
  //     icon: () => <RunsBadge status={workflow?.status} variant="xs" />,
  //     alwaysShowIcon: true,
  //   });

  //   return breadcrumbs;
  // }, [workflow, parentData, selectedTask]);

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
    return (
      tasks &&
      tasks.length > 0 &&
      tasks.some((t) => t.status === V1TaskStatus.RUNNING)
    );
  }, [tasks]);

  const canCancelQueued = useMemo(() => {
    return (
      tasks &&
      tasks.length > 0 &&
      tasks.some((t) => t.status === V1TaskStatus.QUEUED)
    );
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
    const timings: JSX.Element[] = [
      <span key="created" className="flex items-center gap-2">
        <span>Created</span>
        <RelativeDate date={workflow.createdAt} />
      </span>,
      <span key="started" className="flex items-center gap-2">
        <span>Started</span>
        <RelativeDate date={workflow.startedAt} />
      </span>,
      <span key="duration" className="flex items-center gap-2">
        <span>Duration</span>
        <span className="whitespace-nowrap">
          <Duration
            start={workflow.startedAt}
            end={workflow.finishedAt}
            status={workflow.status}
          />
        </span>
        {lastRefetchTime}
      </span>,
    ];

    const interleavedTimings: JSX.Element[] = [];
    timings.forEach((timing, index) => {
      interleavedTimings.push(timing);
      if (index < timings.length - 1) {
        interleavedTimings.push(
          <span key={`sep-${index}`} className="text-sm text-muted-foreground">
            |
          </span>,
        );
      }
    });

    return (
      <span className="flex flex-col items-end sm:flex-row sm:items-center sm:justify-start gap-x-4 gap-y-2 text-sm text-muted-foreground">
        {interleavedTimings}
      </span>
    );
  };

  const taskCount = tasks?.length || 0;

  return (
    <SheetViewLayout
      sheet={
        <RunDetailSheet
          isOpen={!!taskId}
          onClose={handleCloseSheet}
          workflowRunId={workflowRunId || ''}
          taskId={taskId || ''}
        />
      }
    >
      <Headline>
        <PageTitle description={<Timing />}>
          <div className="text-2xl font-bold truncate flex items-center gap-2">
            <RunsBadge status={workflow.status} variant="xs" />
            <RunId wfRun={workflow} />
          </div>
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
              size="sm"
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
              Cancel
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
              size="sm"
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
              Replay
            </SplitButton>
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      <Separator className="my-4" />

      <Tabs
        defaultValue={taskCount > 1 ? 'minimap' : 'waterfall'}
        className="w-full"
      >
        <TabsList layout="underlined" className="w-full">
          {taskCount > 1 && (
            <TabsTrigger variant="underlined" value="minimap">
              Minimap
            </TabsTrigger>
          )}
          <TabsTrigger variant="underlined" value="waterfall">
            Waterfall
          </TabsTrigger>
        </TabsList>
        <TabsContent value="minimap" className="mt-4">
          {taskCount > 1 && workflowRunId && (
            <div className="w-full overflow-x-auto bg-slate-100 dark:bg-slate-900">
              <WorkflowRunVisualizer
                workflowRunId={workflowRunId}
                onTaskSelect={handleTaskSelect}
              />
            </div>
          )}
        </TabsContent>
        <TabsContent value="waterfall" className="mt-4">
          <Waterfall
            workflowRunId={workflowRunId!}
            handleTaskSelect={handleTaskSelect}
          />
        </TabsContent>
      </Tabs>

      <div className="grid grid-cols-1 gap-4 mt-8">
        <Tabs defaultValue="activity" className="w-full">
          <TabsList layout="underlined" className="w-full">
            <TabsTrigger variant="underlined" value="activity">
              Activity
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="config">
              Config
            </TabsTrigger>
          </TabsList>
          <TabsContent value="activity" className="mt-4">
            <FilterProvider>
              <RunEventLog
                workflow={workflow}
                onTaskSelect={(taskId, options) => {
                  navigate(
                    ROUTES.runs.taskDetail(workflowRunId!, taskId, options),
                  );
                }}
              />
            </FilterProvider>
          </TabsContent>
          <TabsContent value="config" className="mt-4">
            <WorkflowDetailsProvider workflowId={workflow.workflowId}>
              <WorkflowGeneralSettings />
            </WorkflowDetailsProvider>
          </TabsContent>
        </Tabs>
      </div>

      <TriggerRunModal
        show={showTriggerModal}
        onClose={() => setShowTriggerModal(false)}
        defaultWorkflowId={workflow.workflowId}
        defaultRunId={workflow.metadata.id}
      />
    </SheetViewLayout>
  );
}
