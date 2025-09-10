import { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/v1/ui/badge';
import { V1Event } from '@/lib/api';
import { Button } from '@/components/v1/ui/button';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { useState } from 'react';
import { AdditionalMetadata } from './additional-metadata';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { RunsTable } from '../../workflow-runs-v1/components/runs-table';
import { RunsProvider } from '../../workflow-runs-v1/hooks/runs-provider';

export const EventColumn = {
  id: 'ID',
  key: 'Event',
  seenAt: 'Seen at',
  workflowId: 'Workflow',
  status: 'Status',
  runs: 'Runs',
  metadata: 'Metadata',
  payload: 'Payload',
  scope: 'Scope',
};

export type EventColumnKeys = keyof typeof EventColumn;

export const idKey: EventColumnKeys = 'id';
export const keyKey: EventColumnKeys = 'key';
export const seenAtKey: EventColumnKeys = 'seenAt';
export const workflowKey: EventColumnKeys = 'workflowId';
export const statusKey: EventColumnKeys = 'status';
export const runsKey: EventColumnKeys = 'runs';
export const metadataKey: EventColumnKeys = 'metadata';
export const payloadKey: EventColumnKeys = 'payload';
export const scopeKey: EventColumnKeys = 'scope';

export const columns = ({
  onRowClick,
  openMetadataPopover,
  setOpenMetadataPopover,
  openPayloadPopover,
  setOpenPayloadPopover,
}: {
  onRowClick?: (row: V1Event) => void;
  openMetadataPopover: string | null;
  setOpenMetadataPopover: (id: string | null) => void;
  openPayloadPopover: string | null;
  setOpenPayloadPopover: (id: string | null) => void;
}): ColumnDef<V1Event>[] => {
  return [
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.id} />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="w-fit cursor-pointer pl-0"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            {row.original.metadata.id}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: keyKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.key} />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="w-fit cursor-pointer pl-0"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            {row.original.key}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: seenAtKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.seenAt} />
      ),
      cell: ({ row }) => {
        return (
          <div>
            <RelativeDate date={row.original.metadata.createdAt} />
          </div>
        );
      },
      enableSorting: false,
    },
    // empty columns to get column filtering to work properly
    {
      accessorKey: workflowKey,
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: statusKey,
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: runsKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.runs} />
      ),
      cell: ({ row }) => {
        if (!row.original.workflowRunSummary) {
          return <div>None</div>;
        }

        return <WorkflowRunSummary event={row.original} />;
      },
      enableSorting: false,
    },
    {
      accessorKey: metadataKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.metadata} />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.additionalMetadata}
            isOpen={openMetadataPopover === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setOpenMetadataPopover(row.original.metadata.id);
              } else {
                setOpenMetadataPopover(null);
              }
            }}
            title="Metadata"
            align="end"
          />
        );
      },
      enableSorting: false,
    },
    {
      accessorKey: payloadKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.payload} />
      ),
      cell: ({ row }) => {
        if (!row.original.payload) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.payload}
            isOpen={openPayloadPopover === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setOpenPayloadPopover(row.original.metadata.id);
              } else {
                setOpenPayloadPopover(null);
              }
            }}
            title="Payload"
            align="start"
          />
        );
      },
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: scopeKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={EventColumn.scope} />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.scope}</div>,
      enableSorting: false,
      enableHiding: true,
    },
  ];
};

// eslint-disable-next-line react-refresh/only-export-components
function WorkflowRunSummary({ event }: { event: V1Event }) {
  const [hoverCardOpen, setPopoverOpen] = useState<
    'failed' | 'succeeded' | 'running' | 'queued' | 'cancelled'
  >();

  const numFailed = event.workflowRunSummary?.failed || 0;
  const numSucceeded = event.workflowRunSummary?.succeeded || 0;
  const numRunning = event.workflowRunSummary?.running || 0;
  const numCancelled = event.workflowRunSummary?.cancelled || 0;
  const numQueued = event.workflowRunSummary?.queued || 0;

  const hoverCardContent = (
    <div className="min-w-fit z-40 p-4 bg-white/10 rounded">
      <RunsProvider
        tableKey={`event-runs-${event.metadata.id}`}
        display={{
          hideCounts: true,
          hideMetrics: true,
          hideDateFilter: true,
          hideTriggerRunButton: true,
          hideCancelAndReplayButtons: true,
        }}
        runFilters={{
          triggeringEventExternalId: event.metadata.id,
        }}
      >
        <RunsTable headerClassName="bg-slate-700" />
      </RunsProvider>
    </div>
  );

  return (
    <div className="flex flex-row gap-2 items-center justify-start">
      {numFailed > 0 && (
        <Popover
          open={hoverCardOpen == 'failed'}
          // open={true}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="failed"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('failed')}
            >
              {numFailed} Failed
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numSucceeded > 0 && (
        <Popover
          open={hoverCardOpen == 'succeeded'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="successful"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('succeeded')}
            >
              {numSucceeded} Succeeded
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numRunning > 0 && (
        <Popover
          open={hoverCardOpen == 'running'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="inProgress"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('running')}
            >
              {numRunning} Running
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numCancelled > 0 && (
        <Popover
          open={hoverCardOpen == 'cancelled'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="inProgress"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('cancelled')}
            >
              {numCancelled} Cancelled
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
      {numQueued > 0 && (
        <Popover
          open={hoverCardOpen == 'queued'}
          onOpenChange={(open) => {
            if (!open) {
              setPopoverOpen(undefined);
            }
          }}
        >
          <PopoverTrigger>
            <Badge
              variant="inProgress"
              className="cursor-pointer"
              onClick={() => setPopoverOpen('queued')}
            >
              {numQueued} Queued
            </Badge>
          </PopoverTrigger>
          <PopoverContent
            className="min-w-fit p-0 bg-background border-none z-40"
            align="end"
          >
            {hoverCardContent}
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}
