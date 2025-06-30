import { createColumnHelper } from '@tanstack/react-table';
import { V1TaskEventType, V1TaskEvent, StepRunEventSeverity } from '@/lib/api';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import {
  ArrowLeftEndOnRectangleIcon,
  ServerStackIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { cn } from '@/lib/utils';
import { Link } from 'react-router-dom';
import { Button } from '@/components/v1/ui/button';
import { useRef, useState } from 'react';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import StepRunError from './step-run-detail/step-run-error';

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

const columnHelper = createColumnHelper<V1TaskEvent>();

export const columns = ({
  onRowClick,
  taskDisplayName,
}: {
  onRowClick: (row: V1TaskEvent) => void;
  taskDisplayName: string;
}) => {
  return [
    columnHelper.accessor((row) => row.id, {
      id: 'task',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Task" />
      ),
      cell: ({ row }) => {
        return (
          <div className="min-w-[120px] max-w-[180px]">
            <Badge
              className="cursor-pointer text-xs font-mono py-1 bg-[#ffffff] dark:bg-[#050c1c] border-[#050c1c] dark:border-gray-400"
              variant="outline"
              onClick={() => onRowClick(row.original)}
            >
              <ArrowLeftEndOnRectangleIcon className="w-4 h-4 mr-1" />
              <div className="truncate max-w-[150px]">{taskDisplayName}</div>
            </Badge>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    }),
    columnHelper.accessor((row) => row.timestamp, {
      id: 'timestamp',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Seen at" />
      ),
      cell: ({ row }) => (
        <div className="w-fit min-w-[120px]">
          <RelativeDate date={row.original.timestamp} />
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    }),
    columnHelper.accessor((row) => eventTypeToSeverity(row.eventType), {
      id: 'event',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Event" />
      ),
      cell: ({ row }) => {
        const event = row.original;
        const severity = eventTypeToSeverity(event.eventType);

        return (
          <div className="flex flex-row items-center gap-2">
            <EventIndicator severity={severity} />
            <div className="tracking-wide text-sm flex flex-row gap-4">
              {mapEventTypeToTitle(event.eventType)}
            </div>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    }),
    columnHelper.accessor((row) => row.workerId, {
      id: 'description',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => {
        const items: JSX.Element[] = [];
        const event = row.original;

        if (event.eventType === V1TaskEventType.FAILED) {
          items.push(<ErrorWithHoverCard event={row.original} />);
        }

        if (event.workerId) {
          items.push(
            <Link to={`/v1/workers/${event.workerId}`}>
              <Button
                variant="link"
                size="xs"
                className="font-mono text-xs text-muted-foreground tracking-tight brightness-150"
              >
                <ServerStackIcon className="w-4 h-4 mr-1" />
                View Worker
              </Button>
            </Link>,
          );
        }

        return (
          <div>
            <div className="text-xs text-muted-foreground font-mono tracking-tight">
              {event.message}
            </div>
            {items.length > 0 && (
              <div className="flex flex-col gap-2 mt-2">{items}</div>
            )}
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    }),
  ];
};

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
    case undefined:
      return 'Unknown';
    default:
      // eslint-disable-next-line no-case-declarations
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
        'rounded-full h-[6px] w-[6px]',
      )}
    />
  );
}

function ErrorWithHoverCard({ event }: { event: V1TaskEvent }) {
  const [popoverOpen, setPopoverOpen] = useState(false);

  // containerRef needed due to https://github.com/radix-ui/primitives/issues/1159#issuecomment-2105108943
  const containerRef = useRef<HTMLDivElement>(null);

  return (
    <div ref={containerRef}>
      <Popover
        open={popoverOpen}
        onOpenChange={(open) => {
          if (!open) {
            setPopoverOpen(false);
          }
        }}
      >
        <PopoverTrigger
          onClick={() => {
            if (popoverOpen) {
              return;
            }

            setPopoverOpen(true);
          }}
          className="cursor-pointer"
        >
          <Button
            variant="link"
            size="xs"
            className="font-mono text-xs text-muted-foreground tracking-tight brightness-150"
          >
            <XCircleIcon className="w-4 h-4 mr-1" />
            View Error
          </Button>
        </PopoverTrigger>
        <PopoverContent
          className="min-w-fit p-0 bg-background border-none z-[80]"
          align="start"
          container={containerRef.current}
        >
          <ErrorHoverContents event={event} />
        </PopoverContent>
      </Popover>
    </div>
  );
}

function ErrorHoverContents({ event }: { event: V1TaskEvent }) {
  const errorText = event.errorMessage;

  if (!errorText) {
    return <StepRunError text="No error message found" />;
  }

  return <StepRunError text={errorText} />;
}
