import { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/v1/ui/badge';
import { Checkbox } from '@/components/v1/ui/checkbox';
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
import { TaskRunsTable } from '../../workflow-runs-v1/components/task-runs-table';

export const columns = ({
  onRowClick,
  hoveredEventId,
  setHoveredEventId,
}: {
  onRowClick?: (row: V1Event) => void;
  hoveredEventId: string | null;
  setHoveredEventId: (id: string | null) => void;
}): ColumnDef<V1Event>[] => {
  return [
    {
      id: 'select',
      header: ({ table }) => (
        <Checkbox
          checked={
            table.getIsAllPageRowsSelected() ||
            (table.getIsSomePageRowsSelected() && 'indeterminate')
          }
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
          className="translate-y-[2px]"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          className="translate-y-[2px]"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'EventId',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Event Id" />
      ),
      cell: ({ row }) => (
        <div className="w-full">{row.original.metadata.id}</div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'key',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Event" />
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
            {row.getValue('key')}
          </Button>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'Seen at',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Seen at" />
      ),
      cell: ({ row }) => {
        return (
          <div>
            <RelativeDate date={row.original.metadata.createdAt} />
          </div>
        );
      },
    },
    // empty columns to get column filtering to work properly
    {
      accessorKey: 'workflows',
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: 'status',
      header: () => <></>,
      cell: () => {
        return <div></div>;
      },
    },
    {
      accessorKey: 'Runs',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Runs" />
      ),
      cell: ({ row }) => {
        if (!row.original.workflowRunSummary) {
          return <div>None</div>;
        }

        return <WorkflowRunSummary event={row.original} />;
      },
    },
    {
      accessorKey: 'Metadata',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Metadata" />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.additionalMetadata}
            isOpen={hoveredEventId === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setHoveredEventId(row.original.metadata.id);
              } else {
                setHoveredEventId(null);
              }
            }}
          />
        );
      },
      enableSorting: false,
    },
    {
      accessorKey: 'Payload',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Payload" />
      ),
      cell: ({ row }) => {
        if (!row.original.payload) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.payload}
            isOpen={hoveredEventId === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setHoveredEventId(row.original.metadata.id);
              } else {
                setHoveredEventId(null);
              }
            }}
          />
        );
      },
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'scope',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Scope" />
      ),
      cell: ({ row }) => <div className="w-full">{row.getValue('scope')}</div>,
      enableSorting: false,
      enableHiding: true,
    }, // {
    //   id: "actions",
    //   cell: ({ row }) => <DataTableRowActions row={row} labels={[]} />,
    // },
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
      <TaskRunsTable
        triggeringEventExternalId={event.metadata.id}
        showCounts={false}
        showMetrics={false}
        showDateFilter={false}
        showTriggerRunButton={false}
        headerClassName="bg-slate-700"
      />
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
