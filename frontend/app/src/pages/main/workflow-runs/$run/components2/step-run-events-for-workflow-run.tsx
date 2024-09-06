import {
  StepRun,
  StepRunEvent,
  StepRunEventReason,
  StepRunEventSeverity,
  StepRunArchive,
  queries,
  WorkflowRunShape,
  Step,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Spinner } from '@/components/ui/loading';
import RelativeDate from '@/components/molecules/relative-date';
import { Button } from '@/components/ui/button';
import { ArrowRightIcon, ChevronRightIcon } from '@radix-ui/react-icons';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/utils';
import React, { useMemo, useState } from 'react';
import {
  SemaphoreEventData,
  SemaphoreExtra,
  columns,
  mapSemaphoreExtra,
} from './step-runs-worker-label-columns';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { Badge } from '@/components/ui/badge';

export function StepRunEvents({
  workflowRun,
  filteredStepRunId,
  onClick,
}: {
  workflowRun: WorkflowRunShape;
  filteredStepRunId?: string;
  onClick: (stepRunId?: string) => void;
}) {
  // TODO update only new things lastId
  const eventsQuery = useQuery({
    ...queries.workflowRuns.listStepRunEvents(
      workflowRun.tenantId,
      workflowRun.metadata.id,
    ),
    refetchInterval: () => {
      // if (workflowRun.status === StepRunStatus.RUNNING) {
      //   return 1000;
      // }

      return 1000;
    },
  });

  const filteredEvents = useMemo(() => {
    if (!filteredStepRunId) {
      return eventsQuery.data?.rows || [];
    }

    return eventsQuery.data?.rows?.filter(
      (x) => x.stepRunId === filteredStepRunId,
    );
  }, [eventsQuery.data, filteredStepRunId]);

  const stepRuns = useMemo(() => {
    return (
      workflowRun.jobRuns?.flatMap((jr) => jr.stepRuns).filter((x) => !!x) ||
      ([] as StepRun[])
    );
  }, [workflowRun]);

  const steps = useMemo(() => {
    return (
      (
        workflowRun.jobRuns
          ?.flatMap((jr) => jr.job?.steps)
          .filter((x) => !!x) || ([] as Step[])
      ).flatMap((x) => x) || ([] as Step[])
    );
  }, [workflowRun]);

  const normalizedStepRunsByStepRunId = useMemo(() => {
    return stepRuns.reduce(
      (acc, stepRun) => {
        acc[stepRun.metadata.id] = stepRun;
        return acc;
      },
      {} as Record<string, StepRun>,
    );
  }, [stepRuns]);

  const normalizedStepsByStepRunId = useMemo(() => {
    return stepRuns.reduce(
      (acc, stepRun) => {
        const step = steps?.find((s) => s.metadata.id === stepRun.stepId);
        if (step) {
          acc[stepRun.metadata.id] = step;
        }
        return acc;
      },
      {} as Record<string, Step>,
    );
  }, [steps, stepRuns]);

  return (
    <div className="flex flex-col gap-2">
      {eventsQuery.isLoading && <Spinner />}
      {eventsQuery.data?.rows?.length === 0 && (
        <Card className="bg-muted/30 h-[400px]">
          <CardHeader>
            <CardTitle className="tracking-wide text-sm">
              No events found
            </CardTitle>
          </CardHeader>
        </Card>
      )}
      {filteredEvents?.map((item, index) => (
        <StepRunEventCard
          key={index}
          event={item}
          stepRun={normalizedStepRunsByStepRunId[item.stepRunId]}
          step={normalizedStepsByStepRunId[item.stepRunId]}
          onClick={onClick}
        />
      ))}
    </div>
  );
}

function StepRunEventCard({
  event,
  stepRun,
  step,
  onClick,
}: {
  event: StepRunEvent;
  stepRun?: StepRun;
  step?: Step;
  onClick: (stepRunId?: string) => void;
}) {
  return (
    <Card className="bg-muted/30">
      <CardHeader>
        <div className="flex flex-row justify-between items-center text-sm">
          <div className="flex flex-row justify-between gap-3 items-center">
            <EventIndicator severity={event.severity} />
            <CardTitle className="tracking-wide text-sm flex flex-row gap-4">
              <Badge
                className="cursor-pointer"
                onClick={() => onClick(stepRun?.metadata.id)}
              >
                {step?.readableId}
              </Badge>
              {getTitleFromReason(event.reason, event.message)}
            </CardTitle>
          </div>
          <RelativeDate date={event.timeLastSeen} />
        </div>
        <CardDescription className="mt-2">{event.message}</CardDescription>
      </CardHeader>
      <CardContent className="p-0 z-10 ">
        <RenderSemaphoreExtra event={event} />
      </CardContent>
      {renderCardFooter(event)}
    </Card>
  );
}

function StepRunArchiveCard({ archive }: { archive: StepRunArchive }) {
  const [isCollapsed, setIsCollapsed] = useState(true);

  const reason = archive.cancelledReason || archive.error || archive.output;
  if (!reason) {
    return <></>;
  }

  const type = archive.cancelledReason
    ? 'Cancelled'
    : archive.error
      ? 'Error'
      : 'Output';

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed);
  };

  return (
    <Card
      className="bg-muted/30 cursor-pointer hover:bg-muted/50 transition-all"
      onClick={toggleCollapse}
    >
      <CardHeader>
        <div className="flex flex-row justify-between items-center text-sm">
          <div className="flex flex-row justify-between gap-3 items-center">
            <span
              className={`transform transition-transform ${isCollapsed ? '' : 'rotate-90'}`}
            >
              <ChevronRightIcon className="w-4 h-4" />
            </span>
            <CardTitle className="tracking-wide text-sm">{type}</CardTitle>
          </div>
          <div className="flex flex-row items-center gap-1">
            <RelativeDate date={archive.createdAt} />
          </div>
        </div>
        {!isCollapsed && (
          <CardDescription className="pt-2">
            <code>{reason}</code>
          </CardDescription>
        )}
      </CardHeader>
      {!isCollapsed && (
        <CardContent className="p-0 z-10 bg-background"></CardContent>
      )}
    </Card>
  );
}

const REASON_TO_TITLE: Record<StepRunEventReason, string> = {
  [StepRunEventReason.ASSIGNED]: 'Assigned to worker',
  [StepRunEventReason.STARTED]: 'Started',
  [StepRunEventReason.FINISHED]: 'Completed',
  [StepRunEventReason.FAILED]: 'Failed',
  [StepRunEventReason.CANCELLED]: 'Cancelled',
  [StepRunEventReason.RETRYING]: 'Retrying',
  [StepRunEventReason.REQUEUED_NO_WORKER]: 'Requeueing (no worker available)',
  [StepRunEventReason.REQUEUED_RATE_LIMIT]: 'Requeueing (rate limit)',
  [StepRunEventReason.SCHEDULING_TIMED_OUT]: 'Scheduling timed out',
  [StepRunEventReason.TIMEOUT_REFRESHED]: 'Timeout refreshed',
  [StepRunEventReason.REASSIGNED]: 'Reassigned',
  [StepRunEventReason.TIMED_OUT]: 'Execution timed out',
  [StepRunEventReason.SLOT_RELEASED]: 'Slot released',
  [StepRunEventReason.RETRIED_BY_USER]: 'Replayed by user',
};

function getTitleFromReason(reason: StepRunEventReason, message: string) {
  return REASON_TO_TITLE[reason] || message;
}

const RenderSemaphoreExtra: React.FC<{ event: StepRunEvent }> = ({ event }) => {
  const state = useMemo(() => {
    const data = (event?.data as { semaphore?: SemaphoreEventData })?.semaphore;
    if (!data) {
      return;
    }

    return mapSemaphoreExtra(data) as unknown as SemaphoreExtra[];
  }, [event]);

  const [isCollapsed, setIsCollapsed] = useState(true);

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed);
  };

  if (!state) {
    return <></>;
  } else {
    return (
      <div className="flex flex-col px-2 gap mb-8">
        <div
          className="flex flex-row gap-3 items-center cursor-pointer hover:bg-muted/50 transition-all"
          onClick={toggleCollapse}
        >
          <span
            className={`transform transition-transform ${isCollapsed ? '' : 'rotate-90'}`}
          >
            <ChevronRightIcon className="w-4 h-4" />
          </span>{' '}
          Desired Worker Labels
        </div>
        {!isCollapsed && (
          <div className="text-sm">
            <DataTable columns={columns} data={state} filters={[]} />
          </div>
        )}
      </div>
    );
  }
};

function renderCardFooter(event: StepRunEvent) {
  if (event.data) {
    const data = event.data as any;

    if (data.worker_id) {
      return (
        <CardFooter>
          <Link to={`/workers/${data.worker_id}`}>
            <Button variant="link" size="xs">
              <ArrowRightIcon className="w-4 h-4 mr-1" />
              View Worker
            </Button>
          </Link>
        </CardFooter>
      );
    }
  }

  return null;
}

const RUN_STATUS_VARIANTS: Record<StepRunEventSeverity, string> = {
  INFO: 'border-transparent rounded-full bg-green-500',
  CRITICAL: 'border-transparent rounded-full bg-red-500',
  WARNING: 'border-transparent rounded-full bg-yellow-500',
};

function EventIndicator({ severity }: { severity: StepRunEventSeverity }) {
  return (
    <div
      className={cn(
        RUN_STATUS_VARIANTS[severity],
        'rounded-full h-[6px] w-[6px]',
      )}
    />
  );
}

function createStepRunArchive(stepRun: StepRun, order: number): StepRunArchive {
  return {
    createdAt: stepRun.finishedAt || stepRun.cancelledAt || stepRun.startedAt!,
    stepRunId: stepRun.metadata.id,
    order: order,
    input: stepRun.input,
    output: stepRun.output,
    startedAt: stepRun.startedAt,
    error: stepRun.error,
    startedAtEpoch: stepRun.startedAtEpoch,
    finishedAt: stepRun.finishedAt,
    finishedAtEpoch: stepRun.finishedAtEpoch,
    timeoutAt: stepRun.timeoutAt,
    timeoutAtEpoch: stepRun.timeoutAtEpoch,
    cancelledAt: stepRun.cancelledAt,
    cancelledAtEpoch: stepRun.cancelledAtEpoch,
    cancelledReason: stepRun.cancelledReason,
    cancelledError: stepRun.cancelledError,
  };
}
