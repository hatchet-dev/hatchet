import { EventWithMetadata } from './step-run-events-for-workflow-run';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { V1TaskEventType, V1TaskEvent, StepRunEventSeverity } from '@/lib/api';
import { cn, emptyGolangUUID } from '@/lib/utils';
import { appRoutes } from '@/router';
import {
  ArrowLeftEndOnRectangleIcon,
  ServerStackIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { Link } from '@tanstack/react-router';

function eventTypeToSeverity(
  eventType: V1TaskEventType | undefined,
): StepRunEventSeverity {
  switch (eventType) {
    case V1TaskEventType.FAILED:
    case V1TaskEventType.RATE_LIMIT_ERROR:
    case V1TaskEventType.SCHEDULING_TIMED_OUT:
    case V1TaskEventType.TIMED_OUT:
    case V1TaskEventType.CANCELLED:
      return StepRunEventSeverity.CRITICAL;
    case V1TaskEventType.REASSIGNED:
    case V1TaskEventType.REQUEUED_NO_WORKER:
    case V1TaskEventType.REQUEUED_RATE_LIMIT:
    case V1TaskEventType.RETRIED_BY_USER:
    case V1TaskEventType.RETRYING:
      return StepRunEventSeverity.WARNING;
    default:
      return StepRunEventSeverity.INFO;
  }
}

export function TaskEventCell({
  event,
  fallbackTaskDisplayName,
  onRowClick,
}: {
  event: EventWithMetadata;
  fallbackTaskDisplayName: string;
  onRowClick: (event: EventWithMetadata) => void;
}) {
  return (
    <div className="min-w-[120px] max-w-[180px]">
      <Badge
        className="cursor-pointer border-[#050c1c] bg-[#ffffff] py-1 font-mono text-xs dark:border-gray-400 dark:bg-[#050c1c]"
        variant="outline"
        onClick={() => onRowClick(event)}
      >
        <ArrowLeftEndOnRectangleIcon className="mr-1 size-4" />
        <div className="max-w-[150px] truncate">
          {event.taskDisplayName || fallbackTaskDisplayName}
        </div>
      </Badge>
    </div>
  );
}

export function TimestampCell({ event }: { event: EventWithMetadata }) {
  return (
    <div className="w-fit min-w-[120px]">
      <RelativeDate date={event.timestamp} />
    </div>
  );
}

export function EventTypeCell({ event }: { event: EventWithMetadata }) {
  const severity = eventTypeToSeverity(event.eventType);

  return (
    <div className="flex flex-row items-center gap-2">
      <EventIndicator severity={severity} />
      <div className="flex flex-row gap-4 text-sm tracking-wide">
        {mapEventTypeToTitle(event.eventType)}
      </div>
    </div>
  );
}

export function DescriptionCell({
  event,
  tenantId,
}: {
  event: EventWithMetadata;
  tenantId: string;
}) {
  const items: JSX.Element[] = [];

  if (event.eventType === V1TaskEventType.FAILED) {
    items.push(<ErrorWithHoverCard key="error" event={event} />);
  }

  if (event.workerId && event.workerId !== emptyGolangUUID) {
    items.push(
      <Link
        to={appRoutes.tenantWorkerRoute.to}
        params={{ tenant: tenantId, worker: event.workerId }}
        key="worker"
      >
        <Button
          variant="link"
          size="xs"
          leftIcon={<ServerStackIcon className="size-4" />}
        >
          View Worker
        </Button>
      </Link>,
    );
  }

  return (
    <div>
      <div
        key="message"
        className="font-mono text-xs tracking-tight text-muted-foreground"
      >
        {event.message}
      </div>
      {items.length > 0 && (
        <div key="items" className="mt-2 flex flex-col items-start gap-2">
          {items}
        </div>
      )}
    </div>
  );
}

function mapEventTypeToTitle(eventType: V1TaskEventType | undefined): string {
  switch (eventType) {
    case V1TaskEventType.ASSIGNED:
      return 'Assigned to worker';
    case V1TaskEventType.STARTED:
      return 'Started';
    case V1TaskEventType.FINISHED:
      return 'Completed';
    case V1TaskEventType.FAILED:
      return 'Failed';
    case V1TaskEventType.CANCELLED:
      return 'Cancelled';
    case V1TaskEventType.RETRYING:
      return 'Retrying';
    case V1TaskEventType.REQUEUED_NO_WORKER:
      return 'Requeuing (no worker available)';
    case V1TaskEventType.REQUEUED_RATE_LIMIT:
      return 'Requeuing (rate limit)';
    case V1TaskEventType.SCHEDULING_TIMED_OUT:
      return 'Scheduling timed out';
    case V1TaskEventType.TIMEOUT_REFRESHED:
      return 'Timeout refreshed';
    case V1TaskEventType.REASSIGNED:
      return 'Reassigned';
    case V1TaskEventType.TIMED_OUT:
      return 'Execution timed out';
    case V1TaskEventType.SLOT_RELEASED:
      return 'Slot released';
    case V1TaskEventType.RETRIED_BY_USER:
      return 'Replayed by user';
    case V1TaskEventType.ACKNOWLEDGED:
      return 'Acknowledged by worker';
    case V1TaskEventType.CREATED:
      return 'Created';
    case V1TaskEventType.RATE_LIMIT_ERROR:
      return 'Rate limit error';
    case V1TaskEventType.SENT_TO_WORKER:
      return 'Sent to worker';
    case V1TaskEventType.QUEUED:
      return 'Queued';
    case V1TaskEventType.SKIPPED:
      return 'Skipped';
    case V1TaskEventType.COULD_NOT_SEND_TO_WORKER:
      return 'Could not send to worker';
    case undefined:
      return 'Unknown';
    default:
      const exhaustiveCheck: never = eventType;
      throw new Error(`Unhandled case: ${exhaustiveCheck}`);
  }
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
        'h-[6px] w-[6px] rounded-full',
      )}
    />
  );
}

function ErrorWithHoverCard({ event }: { event: V1TaskEvent }) {
  return (
    <Popover>
      <PopoverTrigger className="cursor-pointer">
        <Button
          variant="link"
          size="xs"
          leftIcon={<XCircleIcon className="size-4" />}
        >
          View Error
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[300px] max-w-[90vw] border-border bg-popover p-0 shadow-lg sm:w-[400px] md:w-[500px] lg:w-[600px]"
        align="start"
      >
        <div className="w-[300px] max-w-[90vw] p-4 sm:w-[400px] md:w-[500px] lg:w-[600px]">
          <ErrorHoverContents event={event} />
        </div>
      </PopoverContent>
    </Popover>
  );
}

function ErrorHoverContents({ event }: { event: V1TaskEvent }) {
  const errorText = event.errorMessage;

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 border-b border-border pb-2">
        <XCircleIcon className="h-5 w-5 text-destructive" />
        <h3 className="font-medium text-foreground">Error Details</h3>
      </div>
      <div className="h-[400px] overflow-hidden rounded-md border border-border bg-muted/50">
        <div className="scrollbar-thin scrollbar-track-muted scrollbar-thumb-muted-foreground h-full overflow-x-hidden overflow-y-scroll p-4 font-mono text-sm text-foreground">
          <pre className="min-h-[500px] whitespace-pre-wrap break-words">
            {errorText || 'No error message found'}
          </pre>
        </div>
      </div>
    </div>
  );
}
