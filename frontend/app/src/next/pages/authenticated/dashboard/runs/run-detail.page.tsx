import { useParams } from 'react-router-dom';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { AlertCircle } from 'lucide-react';
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import { Skeleton } from '@/next/components/ui/skeleton';
import { useMemo, useCallback, useState } from 'react';
import { RunId } from '@/next/components/runs/run-id';
import { RunsBadge } from '@/next/components/runs/runs-badge';
import { MdOutlineReplay, MdOutlineCancel } from 'react-icons/md';
import { SplitButton } from '@/next/components/ui/split-button';
import {
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
  Headline,
} from '@/next/components/ui/page-header';
import { Duration } from '@/next/components/ui/duration';
import {
  V1TaskStatus,
  V1WorkflowRun,
} from '@/lib/api/generated/data-contracts';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from '@/next/components/ui/tabs';
import { RunEventLog } from '@/next/components/runs/run-event-log/run-event-log';
import { Separator } from '@/next/components/ui/separator';
import { Waterfall } from '@/next/components/waterfall/waterfall';
import RelativeDate from '@/next/components/ui/relative-date';
import { WorkflowDetailsProvider } from '@/next/hooks/use-workflow-details';
import WorkflowGeneralSettings from '../workflows/settings';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { useSidePanel } from '@/next/hooks/use-side-panel';

export default function RunDetailPage() {
  const { workflowRunId } = useParams<{
    workflowRunId: string;
  }>();

  return (
    <RunDetailProvider
      runId={workflowRunId || ''}
      defaultRefetchInterval={1000}
    >
      <RunDetailPageContent workflowRunId={workflowRunId || ''} />
    </RunDetailProvider>
  );
}

type RunDetailPageProps = {
  workflowRunId: string;
  taskId?: string;
};

function RunDetailPageContent({ workflowRunId }: RunDetailPageProps) {
  const { data, isLoading, error, cancel, replay } = useRunDetail();

  const [showTriggerModal, setShowTriggerModal] = useState(false);
  const [selectedTaskId, setSelectedTaskId] = useState<string>();

  const workflow = data?.run;
  const tasks = data?.tasks;

  const { open: openSheet } = useSidePanel();

  const handleTaskSelect = useCallback(
    (taskId: string, childWorkflowRunId?: string) => {
      setSelectedTaskId(taskId);
      openSheet({
        type: 'run-details',
        content: {
          pageWorkflowRunId: workflowRunId,
          selectedWorkflowRunId: childWorkflowRunId || taskId,
          selectedTaskId: taskId,
        },
      });
    },
    [openSheet, workflowRunId],
  );

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

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description={<Timing workflow={workflow} />}>
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
      <Waterfall
        workflowRunId={workflowRunId}
        selectedTaskId={selectedTaskId}
        handleTaskSelect={handleTaskSelect}
      />
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
            <RunEventLog
              workflow={workflow}
              onTaskSelect={(event) => {
                openSheet({
                  type: 'run-details',
                  content: {
                    selectedWorkflowRunId: workflowRunId,
                    selectedTaskId: event.taskId,
                    attempt: event.attempt,
                  },
                });
              }}
            />
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
    </BasicLayout>
  );
}

const Timing = ({ workflow }: { workflow: V1WorkflowRun }) => {
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
    </span>,
  ];
  return (
    <span className="flex flex-col items-start gap-y-2 text-sm text-muted-foreground">
      {timings}
    </span>
  );
};
