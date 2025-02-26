import {
  StepRun,
  StepRunEvent,
  StepRunEventReason,
  StepRunEventSeverity,
  StepRunStatus,
  StepRunArchive,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Button } from '@/components/v1/ui/button';
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
import { DataTable } from '@/components/v1/molecules/data-table/data-table';

export function StepRunEvents({ stepRun }: { stepRun: StepRun | undefined }) {
  const getLogsQuery = useQuery({
    ...queries.stepRuns.listEvents(stepRun?.metadata.id || ''),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const getArchivesQuery = useQuery({
    ...queries.stepRuns.listArchives(stepRun?.metadata.id || ''),
    enabled: !!stepRun,
    refetchInterval: () => {
      if (stepRun?.status === StepRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const combinedEvents = useMemo(() => {
    const events = getLogsQuery.data?.rows || [];
    const stub = createStepRunArchive(stepRun!, 0);

    const archives = getArchivesQuery.data?.rows || [];

    const combined = [
      ...events.map((event) => ({
        type: 'event',
        data: event,
        time: event.timeLastSeen,
      })),
      ...archives.map((archive) => ({
        type: 'archive',
        data: archive,
        time: archive.createdAt,
      })),
    ];

    return [
      {
        type: 'archive',
        data: stub,
        time: stub.createdAt,
      },
      ...combined.sort(
        (a, b) => new Date(b.time).getTime() - new Date(a.time).getTime(),
      ),
    ];
  }, [getLogsQuery.data?.rows, getArchivesQuery.data?.rows, stepRun]);

  if (!stepRun) {
    return <Spinner />;
  }

  return (
    <div className="overflow-y-auto max-h-[400px] flex flex-col gap-2">
      {(getLogsQuery.isLoading || getArchivesQuery.isLoading) && <Spinner />}
      {combinedEvents.length === 0 && (
        <Card className="bg-muted/30 h-[400px]">
          <CardHeader>
            <CardTitle className="tracking-wide text-sm">
              No events found
            </CardTitle>
          </CardHeader>
        </Card>
      )}
      {combinedEvents.map((item, index) =>
        item.type === 'event' ? (
          <StepRunEventCard key={index} event={item.data as StepRunEvent} />
        ) : (
          <StepRunArchiveCard
            key={index}
            archive={item.data as StepRunArchive}
          />
        ),
      )}
    </div>
  );
}

function StepRunEventCard({ event }: { event: StepRunEvent }) {
  return (
    <Card className=" bg-muted/30">
      <CardHeader>
        <div className="flex flex-row justify-between items-center text-sm">
          <div className="flex flex-row justify-between gap-3 items-center">
            <EventIndicator severity={event.severity} />
            <CardTitle className="tracking-wide text-sm">
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
  [StepRunEventReason.ACKNOWLEDGED]: 'Acknowledged by worker',
  [StepRunEventReason.FINISHED]: 'Completed',
  [StepRunEventReason.FAILED]: 'Failed',
  [StepRunEventReason.CANCELLED]: 'Cancelled',
  [StepRunEventReason.RETRYING]: 'Retrying',
  [StepRunEventReason.REQUEUED_NO_WORKER]: 'Requeuing (no worker available)',
  [StepRunEventReason.REQUEUED_RATE_LIMIT]: 'Requeuing (rate limit)',
  [StepRunEventReason.SCHEDULING_TIMED_OUT]: 'Scheduling timed out',
  [StepRunEventReason.TIMEOUT_REFRESHED]: 'Timeout refreshed',
  [StepRunEventReason.REASSIGNED]: 'Reassigned',
  [StepRunEventReason.TIMED_OUT]: 'Execution timed out',
  [StepRunEventReason.SLOT_RELEASED]: 'Slot released',
  [StepRunEventReason.RETRIED_BY_USER]: 'Replayed by user',
  [StepRunEventReason.WORKFLOW_RUN_GROUP_KEY_SUCCEEDED]:
    'Successfully got group key',
  [StepRunEventReason.WORKFLOW_RUN_GROUP_KEY_FAILED]: 'Failed to get group key',
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
    retryCount: 0,
  };
}
