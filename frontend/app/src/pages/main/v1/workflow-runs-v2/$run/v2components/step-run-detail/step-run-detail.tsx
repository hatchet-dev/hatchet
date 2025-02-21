import React from 'react';
import {
  StepRun,
  StepRunStatus,
  V2TaskStatus,
  WorkflowRun,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import invariant from 'tiny-invariant';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { Separator } from '@/components/v1/ui/separator';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import { StepRunEvents } from '../step-run-events-for-workflow-run';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { TaskRunsTable as WorkflowRunsTable } from '../../../components/workflow-runs-table';
import { useTenant } from '@/lib/atoms';
import { V2RunIndicator } from '../../../components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { formatDuration } from '@/lib/utils';
import { V2StepRunOutput } from './step-run-output';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

export enum TabOption {
  Output = 'output',
  ChildWorkflowRuns = 'child-workflow-runs',
  Input = 'input',
  Logs = 'logs',
}

interface StepRunDetailProps {
  taskRunId: string;
  defaultOpenTab?: TabOption;
}

export const STEP_RUN_TERMINAL_STATUSES = [
  StepRunStatus.CANCELLING,
  StepRunStatus.CANCELLED,
  StepRunStatus.FAILED,
  StepRunStatus.SUCCEEDED,
];

const StepRunDetail: React.FC<StepRunDetailProps> = ({
  taskRunId,
  defaultOpenTab = TabOption.Output,
}) => {
  const { tenant } = useTenant();

  const tenantId = tenant?.metadata.id;

  if (!tenantId) {
    throw new Error('Tenant not found');
  }

  const errors: string[] = [];

  const eventsQuery = useQuery({
    ...queries.v2TaskEvents.list(tenantId, taskRunId, {
      offset: 0,
      limit: 50,
    }),
    refetchInterval: () => {
      return 5000;
    },
  });

  const taskRunQuery = useQuery({
    ...queries.v2Tasks.get(taskRunId),
  });

  const events = eventsQuery.data?.rows || [];
  const taskRun = taskRunQuery.data;

  if (eventsQuery.isLoading || taskRunQuery.isLoading) {
    return <Loading />;
  }

  if (events.length === 0 || !taskRun) {
    return <div>No events found</div>;
  }

  return (
    <div className="w-full h-screen overflow-y-scroll flex flex-col gap-4">
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div className="flex flex-row gap-4 items-center">
            {taskRun.status && <V2RunIndicator status={taskRun.status} />}
            <h3 className="text-lg font-mono font-semibold leading-tight tracking-tight text-foreground flex flex-row gap-4 items-center">
              {taskRun.displayName || 'Step Run Detail'}
            </h3>
          </div>
        </div>
      </div>
      <div className="flex flex-row gap-2 items-center">
        <Button
          size={'sm'}
          className="px-2 py-2 gap-2"
          variant={'outline'}
          // disabled={!STEP_RUN_TERMINAL_STATUSES.includes(stepRun.status)}
          onClick={() => {
            // if (!stepRun.input) {
            //   return;
            // }
            // let parsedInput: object;
            // try {
            //   parsedInput = JSON.parse(stepRun.input);
            // } catch (e) {
            //   return;
            // }
            // rerunStepMutation.mutate(parsedInput);
          }}
          disabled
        >
          <ArrowPathIcon className="w-4 h-4" />
          Replay
        </Button>
        <Button
          size={'sm'}
          className="px-2 py-2 gap-2"
          variant={'outline'}
          // disabled={STEP_RUN_TERMINAL_STATUSES.includes(stepRun.status)}
          onClick={() => {
            // cancelStepMutation.mutate();
          }}
          disabled
        >
          <XCircleIcon className="w-4 h-4" />
          Cancel
        </Button>
      </div>
      {errors && errors.length > 0 && (
        <div className="mt-4">
          {errors.map((error, index) => (
            <div key={index} className="text-red-500">
              {error}
            </div>
          ))}
        </div>
      )}
      <div className="flex flex-row gap-2 items-center">
        <V2StepRunSummary taskRunId={taskRunId} />
      </div>
      <Tabs defaultValue={defaultOpenTab}>
        <TabsList layout="underlined">
          <TabsTrigger variant="underlined" value={TabOption.Output}>
            Output
          </TabsTrigger>
          {/* {stepRun.childWorkflowRuns &&
            stepRun.childWorkflowRuns.length > 0 && (
              <TabsTrigger
                variant="underlined"
                value={TabOption.ChildWorkflowRuns}
              >
                Children ({stepRun.childWorkflowRuns.length})
              </TabsTrigger>
            )} */}
          <TabsTrigger variant="underlined" value={TabOption.Input}>
            Input
          </TabsTrigger>
          <TabsTrigger variant="underlined" value={TabOption.Logs}>
            Logs
          </TabsTrigger>
        </TabsList>
        <TabsContent value={TabOption.Output}>
          <V2StepRunOutput taskRunId={taskRunId} />
        </TabsContent>
        <TabsContent value={TabOption.ChildWorkflowRuns}>
          {/* <ChildWorkflowRuns
            stepRun={stepRun}
            workflowRun={workflowRun}
            refetchInterval={5000}
          /> */}
        </TabsContent>
        <TabsContent value={TabOption.Input}>
          {taskRun.input && (
            <CodeHighlighter
              className="my-4 h-[400px] max-h-[400px] overflow-y-auto"
              maxHeight="400px"
              minHeight="400px"
              language="json"
              code={JSON.stringify(JSON.parse(taskRun.input), null, 2)}
            />
          )}
        </TabsContent>
        <TabsContent value={TabOption.Logs}>
          {/* TODO Add this back */}
          {/* <StepRunLogs
            stepRun={stepRun}
            readableId={step?.readableId || 'step'}
          /> */}
        </TabsContent>

        <TabsContent value="logs">App Logs</TabsContent>
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
          taskDisplayName={taskRun.displayName}
        />
      </div>
    </div>
  );
};

export default StepRunDetail;

export const StepRunSummary: React.FC<{ data: StepRun }> = ({ data }) => {
  const timings = [];

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

  if (data.status === StepRunStatus.CANCELLED && data.cancelledAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Cancelled '}
        <RelativeDate date={data.cancelledAt} />
      </div>,
    );
  }

  if (data.status === StepRunStatus.FAILED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Failed '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.status === StepRunStatus.SUCCEEDED && data.finishedAt) {
    timings.push(
      <div key="finished" className="text-sm text-muted-foreground">
        {'Succeeded '}
        <RelativeDate date={data.finishedAt} />
      </div>,
    );
  }

  if (data.finishedAtEpoch && data.startedAtEpoch) {
    timings.push(
      <div key="duration" className="text-sm text-muted-foreground">
        Run took {formatDuration(data.finishedAtEpoch - data.startedAtEpoch)}
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

const V2StepRunSummary = ({ taskRunId }: { taskRunId: string }) => {
  const { tenantId } = useTenant();

  if (!tenantId) {
    throw new Error('Tenant not found');
  }

  const taskRunQuery = useQuery({
    ...queries.v2Tasks.get(taskRunId),
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
  } else {
    timings.push(
      <div key="created" className="text-sm text-muted-foreground">
        Running
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

export function ChildWorkflowRuns({
  stepRun,
  workflowRun,
  refetchInterval,
}: {
  stepRun: StepRun | undefined;
  workflowRun: WorkflowRun;
  refetchInterval?: number;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  return (
    <WorkflowRunsTable
      parentWorkflowRunId={workflowRun.metadata.id}
      parentStepRunId={stepRun?.metadata.id}
      refetchInterval={refetchInterval}
      initColumnVisibility={{
        'Triggered by': false,
      }}
      createdAfter={stepRun?.metadata.createdAt}
    />
  );
}
