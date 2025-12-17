import api, {
  V1TaskStatus,
  V1TaskSummary,
  V1WorkflowRunDetails,
  WorkflowRunShapeForWorkflowRunDetails,
} from '@/lib/api';
import { useParams } from '@tanstack/react-router';
import { WorkflowRunInputDialog } from './v2components/workflow-run-input';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { StepRunEvents } from './v2components/step-run-events-for-workflow-run';
import { useCallback, useRef } from 'react';
import {
  TabOption,
  TaskRunDetail,
} from './v2components/step-run-detail/step-run-detail';
import { Separator } from '@/components/v1/ui/separator';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1RunDetailHeader } from './v2components/header';
import { Badge } from '@/components/v1/ui/badge';
import { ViewToggle } from './v2components/view-toggle';
import WorkflowRunVisualizer from './v2components/workflow-run-visualizer-v2';
import { useAtom } from 'jotai';
import { preferredWorkflowRunViewAtom } from '@/lib/atoms';
import { JobMiniMap } from './v2components/mini-map';
import {
  isTerminalState,
  useWorkflowDetails,
} from '../hooks/use-workflow-details';
import { useQuery } from '@tanstack/react-query';
import { Spinner } from '@/components/v1/ui/loading';
import { Waterfall } from './v2components/waterfall';
import { RunsProvider } from '../hooks/runs-provider';
import { appRoutes } from '@/router';

function statusToBadgeVariant(status: V1TaskStatus) {
  switch (status) {
    case V1TaskStatus.COMPLETED:
      return 'successful';
    case V1TaskStatus.FAILED:
      return 'failed';
    case V1TaskStatus.CANCELLED:
      return 'cancelled';
    case V1TaskStatus.QUEUED:
      return 'queued';
    default:
      return 'inProgress';
  }
}

const GraphView = ({
  shape,
  handleTaskRunExpand,
}: {
  shape: WorkflowRunShapeForWorkflowRunDetails;
  handleTaskRunExpand: (stepRunId: string) => void;
}) => {
  const [view] = useAtom(preferredWorkflowRunViewAtom);

  const showGraphView =
    view == 'graph' && shape.some((task) => task.childrenStepIds.length > 0);

  return showGraphView ? (
    <WorkflowRunVisualizer setSelectedTaskRunId={handleTaskRunExpand} />
  ) : (
    <JobMiniMap
      onClick={(stepRunId) => {
        if (stepRunId) {
          handleTaskRunExpand(stepRunId);
        }
      }}
    />
  );
};

type TaskRunDispatchQueryReturnType = {
  status: V1TaskStatus;
  type: 'task' | 'dag';
  task?: V1TaskSummary;
  dag?: V1WorkflowRunDetails;
};

async function fetchTaskRun(id: string) {
  try {
    return await api.v1TaskGet(id);
  } catch (error) {
    return undefined;
  }
}

async function fetchDAGRun(id: string) {
  try {
    return await api.v1WorkflowRunGet(id);
  } catch (error) {
    return undefined;
  }
}

export default function Run() {
  const { run } = useParams({ from: appRoutes.tenantRunRoute.to });

  const taskRunQuery = useQuery({
    queryKey: ['workflow-run', run],
    queryFn: async (): Promise<TaskRunDispatchQueryReturnType> => {
      const [task, dag] = await Promise.all([
        fetchTaskRun(run),
        fetchDAGRun(run),
      ]);

      if (!task && !dag) {
        throw new Error(`Task or Workflow Run with ID ${run} not found`);
      }

      if (task?.data) {
        const taskData = task.data;

        return {
          status: taskData.status,
          type: 'task',
          task: taskData,
        };
      }

      if (dag?.data?.run) {
        const dagData = dag.data;

        return {
          status: dagData.run.status,
          type: 'dag',
          dag: dagData,
        };
      }

      throw new Error(`Task or Workflow Run with ID ${run} not found`);
    },
    refetchInterval: (query) => {
      const status = query.state.data?.status;

      if (isTerminalState(status)) {
        return 5000;
      }

      return 1000;
    },
  });

  if (taskRunQuery.isLoading) {
    return <Spinner />;
  }

  const runData = taskRunQuery.data;

  if (!runData) {
    return null;
  }

  if (runData.type === 'task') {
    return (
      <RunsProvider tableKey={`task-runs-${run}`}>
        <ExpandedTaskRun id={run} />
      </RunsProvider>
    );
  }

  if (runData.type === 'dag') {
    return (
      <RunsProvider tableKey={`workflow-runs-${run}`}>
        <ExpandedWorkflowRun id={run} />
      </RunsProvider>
    );
  }
}

function ExpandedTaskRun({ id }: { id: string }) {
  return <TaskRunDetail taskRunId={id} defaultOpenTab={TabOption.Output} />;
}

function ExpandedWorkflowRun({ id }: { id: string }) {
  const { open } = useSidePanel();
  const executingRef = useRef(false);

  const handleTaskRunExpand = useCallback(
    (taskRunId: string) => {
      // hack to prevent click handler from firing multiple times,
      // causing index offset issues
      if (executingRef.current) {
        return;
      }

      executingRef.current = true;

      open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Output,
          showViewTaskRunButton: true,
        },
      });

      setTimeout(() => {
        executingRef.current = false;
      }, 100);
    },
    [open],
  );

  const { workflowRun, shape, isLoading, isError } = useWorkflowDetails();

  if (isLoading || isError || !workflowRun) {
    return null;
  }

  const inputData = JSON.stringify(workflowRun.input || {});
  const additionalMetadata = workflowRun.additionalMetadata;

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 pt-2 sm:px-6 lg:px-8">
        <V1RunDetailHeader />
        <Separator className="my-4" />
        <div className="mb-4 flex flex-row gap-x-4">
          <p className="font-semibold">Status</p>
          <Badge variant={statusToBadgeVariant(workflowRun.status)}>
            {workflowRun.status}
          </Badge>
        </div>
        <div className="h-4" />
        <Tabs defaultValue="overview" className="flex h-full flex-col">
          <TabsList layout="underlined" className="mb-4">
            <TabsTrigger variant="underlined" value="overview">
              Overview
            </TabsTrigger>
            <TabsTrigger variant="underlined" value="waterfall">
              Waterfall
            </TabsTrigger>
          </TabsList>
          <TabsContent value="overview" className="min-h-0 flex-1">
            <div className="relative flex h-fit w-full overflow-auto bg-slate-100 dark:bg-slate-900">
              <GraphView
                shape={shape}
                handleTaskRunExpand={handleTaskRunExpand}
              />
              <ViewToggle />
            </div>
            <div className="h-4" />
            <Tabs defaultValue="activity">
              <TabsList layout="underlined">
                <TabsTrigger variant="underlined" value="activity">
                  Activity
                </TabsTrigger>
                <TabsTrigger variant="underlined" value="input">
                  Input
                </TabsTrigger>
                <TabsTrigger variant="underlined" value="additional-metadata">
                  Additional Metadata
                </TabsTrigger>
                {/* <TabsTrigger value="logs">App Logs</TabsTrigger> */}
              </TabsList>
              <TabsContent value="activity">
                <div className="h-4" />
                <StepRunEvents
                  workflowRunId={id}
                  fallbackTaskDisplayName={workflowRun.displayName}
                  onClick={handleTaskRunExpand}
                />
              </TabsContent>
              <TabsContent value="input">
                <WorkflowRunInputDialog input={JSON.parse(inputData)} />
              </TabsContent>
              <TabsContent value="additional-metadata">
                <CodeHighlighter
                  className="my-4"
                  language="json"
                  code={JSON.stringify(additionalMetadata, null, 2)}
                />
              </TabsContent>
            </Tabs>
          </TabsContent>
          <TabsContent value="waterfall" className="min-h-0 flex-1">
            <Waterfall
              workflowRunId={id}
              selectedTaskId={undefined}
              handleTaskSelect={handleTaskRunExpand}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
