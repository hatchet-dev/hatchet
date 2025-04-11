'use client';

import { ColumnDef } from '@tanstack/react-table';
import {
  V1TaskSummary,
  V1TaskStatus,
  WorkflowRunOrderByField,
} from '@/next/lib/api';
import { Time } from '@/next/components/ui/time';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { DataTableColumnHeader } from './data-table-column-header';
import { DataTableRowActions } from './data-table-row-actions';
import { RunId } from '../run-id';
import { RunsBadge } from '../runs-badge';
import { Duration } from '@/next/components/ui/duration';

export const statusOptions = [
  { label: 'Pending', value: 'PENDING' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
];

export const columns: ColumnDef<V1TaskSummary>[] = [
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title="" />,
    cell: ({ row }) => {
      const status = row.getValue('status') as V1TaskStatus;
      return (
        <div className="flex items-center justify-center h-full">
          <RunsBadge variant="xs" status={status} />
        </div>
      );
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'runId',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Run ID" />
    ),
    cell: ({ row }) => <RunId taskRun={row.original} />,
    enableSorting: true,
    enableHiding: false,
  },
  {
    accessorKey: 'startedAt',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Started"
        orderBy={WorkflowRunOrderByField.StartedAt}
      />
    ),
    cell: ({ row }) => {
      const startedAt = row.getValue('startedAt') as string | null;
      if (!startedAt) {
        return <span>-</span>;
      }
      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Time
                  date={startedAt}
                  variant="compact"
                  className="font-mono text-xs text-muted-foreground whitespace-nowrap"
                  asChild
                />
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <Time date={startedAt} variant="timeSince" asChild />
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    },
    enableSorting: true,
    enableHiding: true,
  },

  {
    accessorKey: 'workflowName',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Workflow" />
    ),
    cell: ({ row }) => <div>{row.getValue('workflowName')}</div>,
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'createdAt',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Created"
        orderBy={WorkflowRunOrderByField.CreatedAt}
      />
    ),
    cell: ({ row }) => (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Time date={row.getValue('createdAt')} variant="timeSince" />
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <Time date={row.getValue('createdAt')} variant="timestamp" />
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'duration',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Duration" />
    ),
    cell: ({ row }) => {
      const startedAt = row.getValue('startedAt') as string | null;
      const finishedAt = row.original.finishedAt as string | null;
      const status = row.getValue('status') as V1TaskStatus;

      return (
        <Duration
          start={startedAt}
          end={finishedAt}
          status={status}
          variant="compact"
        />
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    id: 'actions',
    cell: ({ row }) => (
      <div className="flex items-center justify-end h-full">
        <DataTableRowActions row={row} />
      </div>
    ),
    enableHiding: false,
  },
];
