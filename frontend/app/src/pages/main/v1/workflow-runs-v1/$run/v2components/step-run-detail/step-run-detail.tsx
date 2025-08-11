import { V1TaskStatus, V1TaskSummary, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import { LinkIcon } from '@heroicons/react/24/outline';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { Sheet, SheetContent } from '@/components/v1/ui/sheet';
import { StepRunEvents } from '../step-run-events-for-workflow-run';
import { Link } from 'react-router-dom';
import { RunsTable } from '../../../components/runs-table';
import { V1RunIndicator } from '../../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { emptyGolangUUID, formatDuration } from '@/lib/utils';
import { V1StepRunOutput } from './step-run-output';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { TaskRunActionButton } from '@/pages/main/v1/task-runs-v1/actions';
import { TaskRunMiniMap } from '../mini-map';
import { WorkflowDefinitionLink } from '@/pages/main/workflow-runs/$run/v2components/workflow-definition';
import { StepRunLogs } from './step-run-logs';
import { isTerminalState } from '../../../hooks/use-workflow-details';
import { CopyWorkflowConfigButton } from '@/components/v1/shared/copy-workflow-config';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { Waterfall } from '../waterfall';
import { useState, useCallback } from 'react';
import { useToast } from '@/components/v1/hooks/use-toast';
import { Toaster } from '@/components/v1/ui/toaster';

export enum TabOption {
  Output = 'output',
  ChildWorkflowRuns = 'child-workflow-runs',
  Input = 'input',
  Logs = 'logs',
  Waterfall = 'waterfall',
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
  const { tenantId } = useCurrentTenantId();

  if (showViewTaskRunButton) {
    return (
      <Link to={`/tenants/${tenantId}/runs/${taskRun.metadata.id}`}>
        <Button size={'sm'} className="px-2 py-2 gap-2" variant={'outline'}>
          <LinkIcon className="w-4 h-4" />
          View Run
        </Button>
      </Link>
    );
  } else if (
    taskRun.workflowRunExternalId &&
    taskRun.workflowRunExternalId !== emptyGolangUUID &&
    taskRun.workflowRunExternalId !== taskRun.metadata.id
  ) {
    return (
      <Link to={`/tenants/${tenantId}/runs/${taskRun.workflowRunExternalId}`}>
        <Button size={'sm'} className="px-2 py-2 gap-2" variant={'outline'}>
          <LinkIcon className="w-4 h-4" />
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
  const [selectedTaskRunId, setSelectedTaskRunId] = useState<string>();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const { toast } = useToast();

  const handleTaskRunExpand = useCallback((taskRunId: string) => {
    setSelectedTaskRunId(taskRunId);
    setIsSidebarOpen(true);
  }, []);
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

  const onActionProcessed = useCallback(
    (action: 'cancel' | 'replay') => {
      const prefix = action === 'cancel' ? 'Canceling' : 'Replaying';

      const t = toast({
        title: `${prefix} task run`,
        description: `This may take a few seconds. You don't need to hit ${action} again.`,
      });

      setTimeout(() => {
        t.dismiss();
      }, 5000);
    },
    [toast],
  );

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
    <div className="w-full flex flex-col gap-4">
      <Toaster />
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div className="flex flex-row gap-4 items-center">
            {taskRun.status && <V1RunIndicator status={taskRun.status} />}
            <h3 className="text-lg font-mono font-semibold leading-tight tracking-tight text-foreground flex flex-row gap-4 items-center">
              {taskRun.displayName || 'Task Run Detail'}
            </h3>
          </div>
        </div>
      </div>

      {taskRun.parentTaskExternalId && (
        <TriggeringParentWorkflowRunSection
          tenantId={taskRun.tenantId}
          parentTaskExternalId={taskRun.parentTaskExternalId}
        />
      )}
      <div className="flex flex-row gap-2 items-center">
        <TaskRunActionButton
          actionType="replay"
          params={{ externalIds: [taskRunId] }}
          disabled={!TASK_RUN_TERMINAL_STATUSES.includes(taskRun.status)}
          showModal={false}
          onActionProcessed={() => onActionProcessed('replay')}
          onActionSubmit={() => {
            toast({
              title: 'Replay request submitted',
              description: "No need to hit 'Replay' again.",
            });
          }}
        />
        <TaskRunActionButton
          actionType="cancel"
          params={{ externalIds: [taskRunId] }}
          disabled={TASK_RUN_TERMINAL_STATUSES.includes(taskRun.status)}
          showModal={false}
          onActionProcessed={() => onActionProcessed('cancel')}
          onActionSubmit={() => {
            toast({
              title: 'Cancel request submitted',
              description: "No need to hit 'Cancel' again.",
            });
          }}
        />
        <TaskRunPermalinkOrBacklink
          taskRun={taskRun}
          showViewTaskRunButton={showViewTaskRunButton || false}
        />
        <WorkflowDefinitionLink workflowId={taskRun.workflowId} />
        <CopyWorkflowConfigButton workflowConfig={taskRun.workflowConfig} />
      </div>
      <div className="flex flex-row gap-2 items-center">
        <V1StepRunSummary taskRunId={taskRunId} />
      </div>
      <Tabs defaultValue="overview" className="flex flex-col h-full">
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
        <TabsContent value="overview" className="flex-1 min-h-0">
          <div className="w-full h-36 flex relative bg-slate-100 dark:bg-slate-900">
            <TaskRunMiniMap onClick={() => {}} taskRunId={taskRunId} />
          </div>
          <div className="h-4" />
          <Tabs defaultValue={defaultOpenTab}>
            <TabsList layout="underlined">
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
            </TabsList>
            <TabsContent value={TabOption.Output}>
              <V1StepRunOutput taskRunId={taskRunId} />
            </TabsContent>
            <TabsContent value={TabOption.ChildWorkflowRuns} className="mt-4">
              <RunsTable
                tableKey={`child-runs-${taskRunId}`}
                parentTaskExternalId={taskRunId}
                showCounts={false}
                showMetrics={false}
                disableTaskRunPagination={true}
              />
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
              <StepRunLogs taskRun={taskRun} />
            </TabsContent>
          </Tabs>
        </TabsContent>
        {isStandaloneTaskRun && (
          <TabsContent value="waterfall" className="flex-1 min-h-0">
            <Waterfall
              workflowRunId={taskRunId}
              selectedTaskId={selectedTaskRunId}
              handleTaskSelect={handleTaskRunExpand}
            />
          </TabsContent>
        )}
      </Tabs>
      <Separator className="my-4" />
      <div className="mb-8">
        <h3 className="text-lg font-semibold leading-tight text-foreground flex flex-row gap-4 items-center">
          Events
        </h3>
        {/* TODO: Real onclick callback here */}
        <StepRunEvents
          taskRunId={taskRunId}
          onClick={() => {}}
          fallbackTaskDisplayName={taskRun.displayName}
        />
      </div>
      <Sheet
        open={isSidebarOpen}
        onOpenChange={(open) => setIsSidebarOpen(open)}
      >
        <SheetContent className="w-fit min-w-[56rem] max-w-4xl sm:max-w-2xl z-[60] h-full overflow-auto">
          {selectedTaskRunId && (
            <TaskRunDetail
              taskRunId={selectedTaskRunId}
              defaultOpenTab={TabOption.Output}
              showViewTaskRunButton
            />
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
};

const V1StepRunSummary = ({ taskRunId }: { taskRunId: string }) => {
  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
  });

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
    <div className="text-sm text-gray-700 dark:text-gray-300 flex flex-row gap-1">
      Triggered by
      <Link
        to={`/tenants/${tenantId}/runs/${parentWorkflowRun.workflowRunExternalId}`}
        className="font-semibold hover:underline text-indigo-500 dark:text-indigo-200"
      >
        {parentWorkflowRun.displayName} ➶
      </Link>
    </div>
  );
}
