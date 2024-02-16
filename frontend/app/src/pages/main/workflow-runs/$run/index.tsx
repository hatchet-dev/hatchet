import { Separator } from '@/components/ui/separator';
import { StepRun, StepRunStatus, queries, Event } from '@/lib/api';
import CronPrettifier from 'cronstrue';
import { useQuery } from '@tanstack/react-query';
import { Link, useOutletContext, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { Badge } from '@/components/ui/badge';
import { relativeDate, timeBetween } from '@/lib/utils';
import { CodeEditor } from '@/components/ui/code-editor';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import WorkflowRunVisualizer from './components/workflow-run-visualizer';
import { useState } from 'react';
import { StepRunPlayground } from './components/step-run-playground';

export default function ExpandedWorkflowRun() {
  const [selectedStepRun, setSelectedStepRun] = useState<StepRun | undefined>();

  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  const runQuery = useQuery({
    ...queries.workflowRuns.get(tenant.metadata.id, params.run),
    refetchInterval: (query) => {
      return 1000; // FIXME - this is a hack to keep the query running since re-running a step doesn't update the status on the workflow run

      const data = query.state.data;

      if (
        data?.status != 'SUCCEEDED' &&
        data?.status != 'FAILED' &&
        data?.status != 'CANCELLED'
      ) {
        return 1000;
      }
    },
  });

  if (runQuery.isLoading || !runQuery.data) {
    return <Loading />;
  }

  const run = runQuery.data;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <h2 className="text-2xl font-bold leading-tight text-foreground flex flex-row  items-center">
              <Link
                to={`/workflows/${run?.workflowVersion?.workflow?.metadata.id}`}
              >
                {run?.workflowVersion?.workflow?.name}-
                {run?.displayName?.split('-')[1] || run?.metadata.id}
              </Link>
              /{selectedStepRun?.step?.readableId || '*'}
            </h2>
          </div>
          <Badge className="text-sm mt-1" variant={'secondary'}>
            {/* {workflow.versions && workflow.versions[0].version} */}
            {run.status}
          </Badge>
        </div>
        <div className="flex flex-row justify-start items-center gap-2">
          <div className="text-sm text-muted-foreground">
            Created {relativeDate(run?.metadata.createdAt)}
          </div>
          {run?.startedAt && (
            <div className="text-sm text-muted-foreground">
              Started {relativeDate(run.startedAt)}
            </div>
          )}
          {run?.startedAt && run?.finishedAt && (
            <div className="text-sm text-muted-foreground">
              Duration {timeBetween(run.startedAt, run.finishedAt)}
            </div>
          )}
        </div>
        <Separator className="my-4" />
        <div className="w-full h-[150px]">
          <WorkflowRunVisualizer
            workflowRun={run}
            selectedStepRun={selectedStepRun}
            setSelectedStepRun={(step) => {
              setSelectedStepRun(
                step.stepId === selectedStepRun?.stepId ? undefined : step,
              );
            }}
          />
        </div>
        <Separator className="my-4" />
        {!selectedStepRun ? (
          'Select a step to play with'
        ) : (
          <StepRunPlayground
            stepRun={selectedStepRun}
            setStepRun={setSelectedStepRun}
            workflowRun={run}
          />
        )}
      </div>
    </div>
  );
}

export const StepStatusDetails = ({ stepRun }: { stepRun: StepRun }) => {
  let statusText = 'Unknown';

  switch (stepRun.status) {
    case StepRunStatus.RUNNING:
      statusText = 'This step is currently running';
      break;
    case StepRunStatus.FAILED:
      statusText = 'This step failed';

      if (stepRun.error) {
        statusText = `This step failed with error ${stepRun.error}`;
      }

      break;
    case StepRunStatus.CANCELLED:
      statusText = 'This step was cancelled';

      switch (stepRun.cancelledReason) {
        case 'TIMED_OUT':
          statusText = `This step was cancelled because it exceeded its timeout of ${
            stepRun.step?.timeout || '60s'
          }`;
          break;
        case 'SCHEDULING_TIMED_OUT':
          statusText = `This step was cancelled because no workers were available to run ${stepRun.step?.action}`;
          break;
        case 'PREVIOUS_STEP_TIMED_OUT':
          statusText = `This step was cancelled because the previous step timed out`;
          break;
        default:
          break;
      }

      break;
    case StepRunStatus.SUCCEEDED:
      statusText = 'This step succeeded';
      break;
    case StepRunStatus.PENDING:
      statusText = 'This step is pending';
      break;
    default:
      break;
  }

  return statusText;
};

export function StepStatusSection({ stepRun }: { stepRun: StepRun }) {
  const statusText = StepStatusDetails({ stepRun });

  return (
    <div className="mb-4">
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Status
      </h3>
      <div className="text-sm text-muted-foreground">{statusText}</div>
    </div>
  );
}

export function StepDurationSection({ stepRun }: { stepRun: StepRun }) {
  return (
    <div className="mb-4">
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Duration
      </h3>
      <div className="text-sm text-muted-foreground">
        {stepRun.startedAt &&
          stepRun.finishedAt &&
          timeBetween(stepRun.startedAt, stepRun.finishedAt)}
      </div>
    </div>
  );
}

export function StepConfigurationSection({ stepRun }: { stepRun: StepRun }) {
  return (
    <div className="mb-4">
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Configuration
      </h3>
      <div className="text-sm text-muted-foreground">
        Timeout: {stepRun.step?.timeout || '60s'}
      </div>
    </div>
  );
}

function TriggeringEventSection({ event }: { event: Event }) {
  return (
    <>
      <h3 className="text-xl font-semibold leading-tight text-foreground mb-4">
        Triggered by {event.key}
      </h3>
      <EventDataSection event={event} />
    </>
  );
}

function EventDataSection({ event }: { event: Event }) {
  const getEventDataQuery = useQuery({
    ...queries.events.getData(event.metadata.id),
  });

  if (getEventDataQuery.isLoading || !getEventDataQuery.data) {
    return <Loading />;
  }

  const eventData = getEventDataQuery.data;

  return (
    <>
      <CodeEditor
        language="json"
        className="my-4"
        height="400px"
        code={JSON.stringify(JSON.parse(eventData.data), null, 2)}
      />
    </>
  );
}

function TriggeringCronSection({ cron }: { cron: string }) {
  const prettyInterval = `Runs ${CronPrettifier.toString(
    cron,
  ).toLowerCase()} UTC`;

  return (
    <>
      <h3 className="text-xl font-semibold leading-tight text-foreground mb-4">
        Triggered by Cron
      </h3>
      <div className="text-sm text-muted-foreground">{prettyInterval}</div>
      <CodeEditor language="typescript" className="my-4" code={cron} />
    </>
  );
}
