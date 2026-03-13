import { V1RunIndicator } from '../../../components/run-statuses';
import { RunsTable } from '../../../components/runs-table';
import { RunsProvider } from '../../../hooks/runs-provider';
import { useIsTaskRunSkipped } from '../../../hooks/use-is-task-run-skipped';
import { isTerminalState } from '../../../hooks/use-workflow-details';
import { TaskRunMiniMap } from '../mini-map';
import { StepRunEvents } from '../step-run-events-for-workflow-run';
import { Waterfall } from '../waterfall';
import { V1StepRunOutput } from './step-run-output';
import { TaskRunLogs } from './task-run-logs';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Loading } from '@/components/v1/ui/loading';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1TaskStatus, V1TaskSummary, queries } from '@/lib/api';
import { emptyGolangUUID, formatDuration } from '@/lib/utils';
import { TaskRunActionButton } from '@/pages/main/v1/task-runs-v1/actions';
import { WorkflowDefinitionLink } from '@/pages/main/workflow-runs/$run/v2components/workflow-definition';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import { FullscreenIcon } from 'lucide-react';
import { useCallback, useState } from 'react';

export enum TabOption {
  Output = 'output',
  ChildWorkflowRuns = 'child-workflow-runs',
  Input = 'input',
  Logs = 'logs',
  Waterfall = 'waterfall',
  AdditionalMetadata = 'additional-metadata',
  Activity = 'activity',
}

interface TaskRunDetailProps {
  taskRunId: string;
  defaultOpenTab?: TabOption;
  showViewTaskRunButton?: boolean;
}

export const TASK_RUN_TERMINAL_STATUSES = [
  V1TaskStatus.CANCELLED,
  V1TaskStatus.FAILED,
  V1TaskStatus.COMPLETED,
];

const TaskRunPermalinkOrBacklink = ({
  taskRun,
  showViewTaskRunButton,
}: {
  taskRun: V1TaskSummary;
  showViewTaskRunButton: boolean;
}) => {
  const { tenant } = useParams({ from: appRoutes.tenantRoute.to });

  if (showViewTaskRunButton) {
    return (
      <Link
        to={appRoutes.tenantRunRoute.to}
        params={{ tenant: tenant, run: taskRun.metadata.id }}
      >
        <Button
          size={'sm'}
          variant={'outline'}
          leftIcon={<FullscreenIcon className="size-4" />}
        >
          Expand
        </Button>
      </Link>
    );
  } else if (
    taskRun.workflowRunExternalId &&
    taskRun.workflowRunExternalId !== emptyGolangUUID &&
    taskRun.workflowRunExternalId !== taskRun.metadata.id
  ) {
    return (
      <Link
        to={appRoutes.tenantRunRoute.to}
        params={{ tenant: tenant, run: taskRun.workflowRunExternalId }}
      >
        <Button
          size={'sm'}
          variant={'outline'}
          leftIcon={<FullscreenIcon className="size-4" />}
        >
          View DAG Run
        </Button>
      </Link>
    );
  } else {
    return null;
  }
};

export const TaskRunDetail = ({
  taskRunId,
  defaultOpenTab = TabOption.Output,
  showViewTaskRunButton,
}: TaskRunDetailProps) => {
  const { open } = useSidePanel();
  const [logsResetKey, setLogsResetKey] = useState(0);
  const handleTaskRunExpand = useCallback(
    (taskRunId: string) => {
      open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Output,
          showViewTaskRunButton: true,
        },
      });
    },
    [open],
  );
  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;

      if (isTerminalState(status)) {
        return 5000;
      }

      return 1000;
    },
  });

  const { isSkipped } = useIsTaskRunSkipped({ taskRunId });
  const taskRun = taskRunQuery.data;

  if (taskRunQuery.isLoading) {
    return <Loading />;
  }

  if (!taskRun) {
    return <div>No events found</div>;
  }

  const isStandaloneTaskRun =
    taskRun.workflowRunExternalId === emptyGolangUUID ||
    taskRun.workflowRunExternalId === taskRun.metadata.id;

  return (
    <div className="flex w-full flex-col gap-4">
      <div className="flex flex-row items-center justify-between">
        <div className="flex w-full flex-row items-center justify-between">
          <div className="flex flex-row items-center gap-4">
            {taskRun.status && (
              <V1RunIndicator status={taskRun.status} isSkipped={isSkipped} />
            )}
            <h3 className="flex flex-row items-center gap-4 font-mono text-lg font-semibold leading-tight tracking-tight text-foreground">
              {taskRun.displayName || 'Task Run Detail'}
            </h3>
          </div>
        </div>
      </div>

      {taskRun.parentTaskExternalId && (
        <TriggeringParentWorkflowRunSection
          tenant={taskRun.tenantId}
          parentTaskExternalId={taskRun.parentTaskExternalId}
        />
      )}
      <div className="side-responsive-layout flex flex-col items-start justify-start gap-2">
        <div className="side-responsive-inner flex w-full flex-col items-start gap-2">
          <div className="flex flex-row items-center gap-2">
            <RunsProvider tableKey="task-run-detail">
              <TaskRunActionButton
                actionType="replay"
                paramOverrides={{ externalIds: [taskRunId] }}
                disabled={!TASK_RUN_TERMINAL_STATUSES.includes(taskRun.status)}
                showModal={false}
                showLabel
              />
              <TaskRunActionButton
                actionType="cancel"
                paramOverrides={{ externalIds: [taskRunId] }}
                disabled={TASK_RUN_TERMINAL_STATUSES.includes(taskRun.status)}
                showModal={false}
                showLabel
              />
            </RunsProvider>
          </div>
          <div className="flex flex-row items-center gap-2">
            <TaskRunPermalinkOrBacklink
              taskRun={taskRun}
              showViewTaskRunButton={showViewTaskRunButton || false}
            />
            <WorkflowDefinitionLink workflowId={taskRun.workflowId} />
            <CopyWorkflowConfigButton workflowConfig={taskRun.workflowConfig} />
          </div>
        </div>
      </div>
      <div className="flex flex-row items-center gap-2">
        <V1StepRunSummary taskRunId={taskRunId} />
      </div>
      <Tabs defaultValue="overview" className="flex h-full flex-col">
        <TabsList layout="underlined" className="mb-4">
          <TabsTrigger variant="underlined" value="overview">
            Overview
          </TabsTrigger>
          {isStandaloneTaskRun && (
            <TabsTrigger variant="underlined" value="waterfall">
              Waterfall
            </TabsTrigger>
          )}
        </TabsList>
        <TabsContent value="overview" className="min-h-0 flex-1">
          <div className="relative flex w-full bg-slate-100 dark:bg-slate-900">
            <TaskRunMiniMap onClick={() => {}} taskRunId={taskRunId} />
          </div>
          <div className="h-4" />
          <Tabs
            defaultValue={defaultOpenTab}
            onValueChange={(value) => {
              if (value === TabOption.Logs) {
                // Increment counter to force remount when Logs tab is opened
                setLogsResetKey((prev: number) => prev + 1);
              }
            }}
          >
            <TabsList layout="underlined">
              <TabsTrigger variant="underlined" value={TabOption.Activity}>
                Activity
              </TabsTrigger>
              <TabsTrigger variant="underlined" value={TabOption.Output}>
                Output
              </TabsTrigger>
              {taskRun.numSpawnedChildren > 0 && (
                <TabsTrigger
                  variant="underlined"
                  value={TabOption.ChildWorkflowRuns}
                >
                  Children ({taskRun.numSpawnedChildren})
                </TabsTrigger>
              )}
              <TabsTrigger variant="underlined" value={TabOption.Input}>
                Input
              </TabsTrigger>
              <TabsTrigger variant="underlined" value={TabOption.Logs}>
                Logs
              </TabsTrigger>
              <TabsTrigger
                variant="underlined"
                value={TabOption.AdditionalMetadata}
                className="side-responsive-layout"
              >
                <span className="side-responsive-inner flex">
                  <span className="side-sm:hidden block">Metadata</span>
                  <span className="side-sm:block hidden">
                    Additional Metadata
                  </span>
                </span>
              </TabsTrigger>
            </TabsList>
            <TabsContent value={TabOption.Output}>
              <V1StepRunOutput taskRunId={taskRunId} />
            </TabsContent>
            <TabsContent value={TabOption.Activity}>
              <div className="py-4">
                <StepRunEvents
                  taskRunId={taskRunId}
                  fallbackTaskDisplayName={taskRun.displayName}
                />
              </div>
            </TabsContent>
            <TabsContent value={TabOption.ChildWorkflowRuns} className="mt-4">
              <div className="flex flex-col h-96">
                <RunsProvider
                  tableKey={`child-runs-${taskRunId}`}
                  display={{
                    hideCounts: true,
                    hideMetrics: true,
                    hideDateFilter: true,
                    hideTriggerRunButton: true,
                  }}
                  runFilters={{
                    parentTaskExternalId: taskRunId,
                  }}
                >
                  <RunsTable />
                </RunsProvider>
              </div>
            </TabsContent>
            <TabsContent value={TabOption.Input}>
              {taskRun.input && (
                <CodeHighlighter
                  className="my-4 h-[400px] max-h-[400px] overflow-y-auto"
                  maxHeight="400px"
                  minHeight="400px"
                  language="json"
                  code={JSON.stringify(taskRun.input, null, 2)}
                />
              )}
            </TabsContent>
            <TabsContent value={TabOption.Logs}>
              <TaskRunLogs resetTrigger={logsResetKey} taskRun={taskRun} />
            </TabsContent>
            <TabsContent value={TabOption.AdditionalMetadata}>
              <CodeHighlighter
                className="my-4 h-[400px] max-h-[400px] overflow-y-auto"
                maxHeight="400px"
                minHeight="400px"
                language="json"
                code={JSON.stringify(taskRun.additionalMetadata ?? {}, null, 2)}
              />
            </TabsContent>
          </Tabs>
        </TabsContent>
        {isStandaloneTaskRun && (
          <TabsContent value="waterfall" className="min-h-0 flex-1">
            <Waterfall
              workflowRunId={taskRunId}
              selectedTaskId={undefined}
              handleTaskSelect={handleTaskRunExpand}
            />
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
};

const V1StepRunSummary = ({ taskRunId }: { taskRunId: string }) => {
  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
  });
  const { isSkipped: hasSkippedEvent } = useIsTaskRunSkipped({ taskRunId });

  const timings = [];
  const data = taskRunQuery.data;

  if (taskRunQuery.isLoading || !data) {
    return <Loading />;
  }

  if (data.startedAt) {
    timings.push(
      <div key="created" className="text-sm text-muted-foreground">
        {'Started '}
        <RelativeDate date={data.startedAt} />
      </div>,
    );
  }

  if (data.status === V1TaskStatus.FAILED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Failed '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === V1TaskStatus.COMPLETED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {hasSkippedEvent ? 'Skipped ' : 'Succeeded '}
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
    <div className="flex flex-row items-center gap-4">{interleavedTimings}</div>
  );
};

function TriggeringParentWorkflowRunSection({
  tenant,
  parentTaskExternalId,
}: {
  tenant: string;
  parentTaskExternalId: string;
}) {
  // Get the parent task to find the parent workflow run
  const parentTaskQuery = useQuery({
    ...queries.v1Tasks.get(parentTaskExternalId),
  });

  const parentTask = parentTaskQuery.data;
  const parentWorkflowRunId = parentTask?.workflowRunExternalId;

  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(parentTaskExternalId),
  });

  // Show nothing while loading or if no data
  if (parentTaskQuery.isLoading || !parentTask || !parentWorkflowRunId) {
    return null;
  }

  if (taskRunQuery.isLoading || !taskRunQuery.data) {
    return null;
  }

  const parentWorkflowRun = taskRunQuery.data;

  return (
    <div className="flex flex-row gap-1 text-sm text-gray-700 dark:text-gray-300">
      Triggered by
      <Link
        to={appRoutes.tenantRunRoute.to}
        params={{
          tenant: tenant,
          run: parentWorkflowRun.workflowRunExternalId,
        }}
        className="font-semibold text-indigo-500 hover:underline dark:text-indigo-200"
      >
        {parentWorkflowRun.displayName} ➶
      </Link>
    </div>
  );
}
