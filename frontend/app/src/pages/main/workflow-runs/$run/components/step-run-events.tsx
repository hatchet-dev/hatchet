import {
  StepRun,
  StepRunEvent,
  StepRunEventReason,
  StepRunEventSeverity,
  StepRunStatus,
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
} from '@/components/ui/card';
import { Spinner } from '@/components/ui/loading';
import RelativeDate from '@/components/molecules/relative-date';
import { Button } from '@/components/ui/button';
import { ArrowRightIcon } from '@radix-ui/react-icons';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/utils';

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

  if (!stepRun) {
    return <Spinner />;
  }

  return (
    <div className="overflow-y-auto max-h-[400px] flex flex-col gap-2">
      {getLogsQuery.isLoading && <Spinner />}
      {getLogsQuery.data?.rows?.length === 0 && (
        <Card className="bg-muted/30 h-[400px]">
          <CardHeader>
            <CardTitle className="tracking-wide text-sm">
              No events found
            </CardTitle>
          </CardHeader>
        </Card>
      )}
      {getLogsQuery.data?.rows?.map((event) => (
        <StepRunEventCard key={event.id} event={event} />
      ))}
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
      <CardContent className="p-0 z-10 bg-background"></CardContent>
      {renderCardFooter(event)}
    </Card>
  );
}

function getTitleFromReason(reason: StepRunEventReason, message: string) {
  switch (reason) {
    case StepRunEventReason.ASSIGNED:
      return 'Assigned to worker';
    case StepRunEventReason.STARTED:
      return 'Started';
    case StepRunEventReason.FINISHED:
      return 'Completed';
    case StepRunEventReason.FAILED:
      return 'Failed';
    case StepRunEventReason.CANCELLED:
      return 'Cancelled';
    case StepRunEventReason.RETRYING:
      return 'Retrying';
    case StepRunEventReason.REQUEUED_NO_WORKER:
      return 'Requeueing (no worker available)';
    case StepRunEventReason.REQUEUED_RATE_LIMIT:
      return 'Requeueing (rate limit)';
    case StepRunEventReason.SCHEDULING_TIMED_OUT:
      return 'Scheduling timed out';

    default:
      return message;
  }
}

function renderCardFooter(event: StepRunEvent) {
  console.log(event.data);

  if (event.data) {
    const data = event.data as any;

    switch (event.reason) {
      case StepRunEventReason.ASSIGNED:
        // render a link to the worker
        console.log(event.data);
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
