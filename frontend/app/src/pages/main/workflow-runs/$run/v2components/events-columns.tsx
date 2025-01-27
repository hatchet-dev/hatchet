import { createColumnHelper } from '@tanstack/react-table';
import { V2EventType, V2StepRunEvent, StepRunEventSeverity } from '@/lib/api';
import RelativeDate from '@/components/molecules/relative-date';
import { Badge } from '@/components/ui/badge';
import {
  ArrowLeftEndOnRectangleIcon,
  ServerStackIcon,
  XCircleIcon,
} from '@heroicons/react/24/outline';
import { DataTableColumnHeader } from '@/components/molecules/data-table/data-table-column-header';
import { cn } from '@/lib/utils';
import { Link, useOutletContext } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { TenantContextType } from '@/lib/outlet';
import invariant from 'tiny-invariant';
import { useRef, useState } from 'react';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import StepRunError from './step-run-detail/step-run-error';

function eventTypeToSeverity(
  eventType: V2EventType | undefined,
): StepRunEventSeverity {
  switch (eventType) {
    case V2EventType.FAILED:
    case V2EventType.RATE_LIMIT_ERROR:
    case V2EventType.SCHEDULING_TIMED_OUT:
    case V2EventType.TIMED_OUT:
    case V2EventType.CANCELLED:
      return StepRunEventSeverity.CRITICAL;
    case V2EventType.REASSIGNED:
    case V2EventType.REQUEUED_NO_WORKER:
    case V2EventType.REQUEUED_RATE_LIMIT:
    case V2EventType.RETRIED_BY_USER:
    case V2EventType.RETRYING:
      return StepRunEventSeverity.WARNING;
    default:
      return StepRunEventSeverity.INFO;
  }
}

const columnHelper = createColumnHelper<V2StepRunEvent>();

export const columns = ({
  onRowClick,
}: {
  onRowClick: (row: V2StepRunEvent) => void;
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
              <div className="truncate max-w-[150px]">
                {row.original.taskDisplayName}
              </div>
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
    columnHelper.accessor((row) => eventTypeToSeverity(row.event_type), {
      id: 'event',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Event" />
      ),
      cell: ({ row }) => {
        const event = row.original;
        const severity = eventTypeToSeverity(event.event_type);

        return (
          <div className="flex flex-row items-center gap-2">
            <EventIndicator severity={severity} />
            <div className="tracking-wide text-sm flex flex-row gap-4">
              {mapEventTypeToTitle(event.event_type)}
            </div>
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
    }),
    columnHelper.accessor((row) => row.worker_id, {
      id: 'description',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Description" />
      ),
      cell: ({ row }) => {
        const items: JSX.Element[] = [];
        const event = row.original;

        if (event.event_type === V2EventType.FAILED) {
          items.push(<ErrorWithHoverCard event={row.original} rows={[]} />);
        }

        if (event.data) {
          const data = event.data as any;

          if (data.worker_id) {
            items.push(
              <Link to={`/workers/${data.worker_id}`}>
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

function mapEventTypeToTitle(eventType: V2EventType | undefined): string {
  switch (eventType) {
    case V2EventType.ASSIGNED:
      return 'Assigned to worker';
    case V2EventType.STARTED:
      return 'Started';
    case V2EventType.FINISHED:
      return 'Completed';
    case V2EventType.FAILED:
      return 'Failed';
    case V2EventType.CANCELLED:
      return 'Cancelled';
    case V2EventType.RETRYING:
      return 'Retrying';
    case V2EventType.REQUEUED_NO_WORKER:
      return 'Requeuing (no worker available)';
    case V2EventType.REQUEUED_RATE_LIMIT:
      return 'Requeuing (rate limit)';
    case V2EventType.SCHEDULING_TIMED_OUT:
      return 'Scheduling timed out';
    case V2EventType.TIMEOUT_REFRESHED:
      return 'Timeout refreshed';
    case V2EventType.REASSIGNED:
      return 'Reassigned';
    case V2EventType.TIMED_OUT:
      return 'Execution timed out';
    case V2EventType.SLOT_RELEASED:
      return 'Slot released';
    case V2EventType.RETRIED_BY_USER:
      return 'Replayed by user';
    case V2EventType.ACKNOWLEDGED:
      return 'Acknowledged by worker';
    case V2EventType.CREATED:
      return 'Created';
    case V2EventType.RATE_LIMIT_ERROR:
      return 'Rate limit error';
    case V2EventType.SENT_TO_WORKER:
      return 'Sent to worker';
    default:
      return 'Unknown event';
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

function ErrorWithHoverCard({
  event,
  rows,
}: {
  event: V2StepRunEvent;
  rows: V2StepRunEvent[];
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);
  invariant(event.taskId);

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
          <ErrorHoverContents event={event} rows={rows} />
        </PopoverContent>
      </Popover>
    </div>
  );
}

function ErrorHoverContents({
  event,
  rows,
}: {
  event: V2StepRunEvent;
  rows: V2StepRunEvent[];
}) {
  // We cannot call this component without stepRun being defined.
  invariant(event.taskId);

  const errorText = rows
    .filter(
      (row) =>
        row.event_type === V2EventType.FAILED ||
        row.event_type === V2EventType.CANCELLED,
    )
    .sort((a, b) => {
      const lhs = new Date(a.timestamp);
      const rhs = new Date(b.timestamp);

      return lhs.getTime() - rhs.getTime();
    })
    .at(0);

  if (!errorText || !errorText.error_message) {
    return <StepRunError text="No error message found" />;
  }

  return <StepRunError text={errorText.error_message} />;
}
