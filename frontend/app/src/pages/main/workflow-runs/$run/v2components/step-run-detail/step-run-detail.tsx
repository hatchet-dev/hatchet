import React, { useMemo, useState } from 'react';
import api, {
  StepRun,
  StepRunStatus,
  WorkflowRun,
  WorkflowRunShape,
  queries,
} from '@/lib/api';
import StepRunOutput from './step-run-output';
import RelativeDate from '@/components/molecules/relative-date';
import { RunIndicator } from '../../../components/run-statuses';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useApiError } from '@/lib/hooks';
import invariant from 'tiny-invariant';
import { Button } from '@/components/ui/button';
import { Loading } from '@/components/ui/loading';
import { ArrowPathIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { Separator } from '@/components/ui/separator';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { StepRunLogs } from './step-run-logs';
import { CodeHighlighter } from '@/components/ui/code-highlighter';
import { StepRunEvents } from '../step-run-events-for-workflow-run';
import { useOutletContext } from 'react-router-dom';
import { TenantContextType } from '@/lib/outlet';
import { WorkflowRunsTable } from '../../../components/workflow-runs-table';
import { formatDuration } from '@/lib/utils';

export enum TabOption {
  Output = 'output',
  ChildWorkflowRuns = 'child-workflow-runs',
  Input = 'input',
  Logs = 'logs',
}

interface StepRunDetailProps {
  stepRunId: string;
  workflowRun: WorkflowRunShape;
  defaultOpenTab?: TabOption;
}

export const STEP_RUN_TERMINAL_STATUSES = [
  StepRunStatus.CANCELLING,
  StepRunStatus.CANCELLED,
  StepRunStatus.FAILED,
  StepRunStatus.SUCCEEDED,
];

const StepRunDetail: React.FC<StepRunDetailProps> = ({
  stepRunId,
  workflowRun,
  defaultOpenTab = TabOption.Output,
}) => {
  const [errors, setErrors] = useState<string[]>([]);

  const { handleApiError } = useApiError({
    setErrors,
  });

  const queryClient = useQueryClient();

  const getStepRunQuery = useQuery({
    ...queries.stepRuns.get(workflowRun.tenantId, stepRunId),
    refetchInterval: (query) => {
      const data = query.state.data;

      if (data?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const stepRun = getStepRunQuery.data;

  const step = useMemo(() => {
    return workflowRun.jobRuns
      ?.flatMap((jr) => jr.job?.steps)
      .filter((x) => !!x)
      .find((x) => x?.metadata.id === stepRun?.stepId);
  }, [workflowRun, stepRun]);

  const rerunStepMutation = useMutation({
    mutationKey: [
      'step-run:update:rerun',
      stepRun?.tenantId,
      stepRun?.metadata.id,
    ],
    mutationFn: async (input: object) => {
      invariant(stepRun?.tenantId, 'has tenantId');

      const res = await api.stepRunUpdateRerun(
        stepRun?.tenantId,
        stepRun?.metadata.id,
        {
          input,
        },
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (stepRun: StepRun) => {
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.get(
          stepRun?.tenantId,
          workflowRun.metadata.id,
        ).queryKey,
      });
    },
    onError: handleApiError,
  });

  const cancelStepMutation = useMutation({
    mutationKey: [
      'step-run:update:cancel',
      stepRun?.tenantId,
      stepRun?.metadata.id,
    ],
    mutationFn: async () => {
      invariant(stepRun?.tenantId, 'has tenantId');

      const res = await api.stepRunUpdateCancel(
        stepRun?.tenantId,
        stepRun?.metadata.id,
      );

      return res.data;
    },
    onMutate: () => {
      setErrors([]);
    },
    onSuccess: (stepRun: StepRun) => {
      queryClient.invalidateQueries({
        queryKey: queries.workflowRuns.get(
          stepRun?.tenantId,
          workflowRun.metadata.id,
        ).queryKey,
      });

      getStepRunQuery.refetch();
    },
    onError: handleApiError,
  });

  if (!stepRun) {
    return <Loading />;
  }

  return (
    <div className="w-full h-screen overflow-y-scroll flex flex-col gap-4">
      <div className="flex flex-row justify-between items-center">
        <div className="flex flex-row justify-between items-center w-full">
          <div className="flex flex-row gap-4 items-center">
            <RunIndicator status={stepRun.status} />
            <h3 className="text-lg font-mono font-semibold leading-tight tracking-tight text-foreground flex flex-row gap-4 items-center">
              {step?.readableId || 'Step Run Detail'}
            </h3>
          </div>
        </div>
      </div>
      <div className="flex flex-row gap-2 items-center">
        <Button
          size={'sm'}
          className="px-2 py-2 gap-2"
          variant={'outline'}
          disabled={!STEP_RUN_TERMINAL_STATUSES.includes(stepRun.status)}
          onClick={() => {
            if (!stepRun.input) {
              return;
            }

            let parsedInput: object;

            try {
              parsedInput = JSON.parse(stepRun.input);
            } catch (e) {
              return;
            }

            rerunStepMutation.mutate(parsedInput);
          }}
        >
          <ArrowPathIcon className="w-4 h-4" />
          Replay
        </Button>
        <Button
          size={'sm'}
          className="px-2 py-2 gap-2"
          variant={'outline'}
          disabled={STEP_RUN_TERMINAL_STATUSES.includes(stepRun.status)}
          onClick={() => {
            cancelStepMutation.mutate();
          }}
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
        {stepRun && <StepRunSummary data={stepRun} />}
      </div>
      <Tabs defaultValue={defaultOpenTab}>
        <TabsList layout="underlined">
          <TabsTrigger variant="underlined" value={TabOption.Output}>
            Output
          </TabsTrigger>
          {stepRun.childWorkflowRuns &&
            stepRun.childWorkflowRuns.length > 0 && (
              <TabsTrigger
                variant="underlined"
                value={TabOption.ChildWorkflowRuns}
              >
                Children ({stepRun.childWorkflowRuns.length})
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
          <StepRunOutput stepRun={stepRun} workflowRun={workflowRun} />
        </TabsContent>
        <TabsContent value={TabOption.ChildWorkflowRuns}>
          <ChildWorkflowRuns
            stepRun={stepRun}
            workflowRun={workflowRun}
            refetchInterval={5000}
          />
        </TabsContent>
        <TabsContent value={TabOption.Input}>
          {stepRun.input && (
            <CodeHighlighter
              className="my-4 h-[400px] max-h-[400px] overflow-y-auto"
              maxHeight="400px"
              minHeight="400px"
              language="json"
              code={JSON.stringify(JSON.parse(stepRun?.input || '{}'), null, 2)}
            />
          )}
        </TabsContent>
        <TabsContent value={TabOption.Logs}>
          <StepRunLogs
            stepRun={stepRun}
            readableId={step?.readableId || 'step'}
          />
        </TabsContent>

        {/* <TabsContent value="logs">App Logs</TabsContent> */}
      </Tabs>
      <Separator className="my-4" />
      <div className="mb-8">
        <h3 className="text-lg font-semibold leading-tight text-foreground flex flex-row gap-4 items-center">
          Events
        </h3>
        <StepRunEvents taskRunId={stepRunId} />
      </div>
    </div>
  );
};

export default StepRunDetail;

const StepRunSummary: React.FC<{ data: StepRun }> = ({ data }) => {
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
