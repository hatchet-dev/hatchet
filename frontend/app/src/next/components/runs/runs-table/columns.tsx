'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import {
  V1TaskSummary,
  V1TaskStatus,
  WorkflowRunOrderByField,
} from '@/lib/api';
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
import { AdditionalMetadata } from '@/next/components/ui/additional-meta';
import { useFilters } from '@/next/hooks/use-filters';
import { RunsFilters } from '@/next/hooks/use-runs';

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
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'workflowName',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Definition" />
    ),
    cell: ({ row }) => <div>{row.getValue('workflowName')}</div>,
    enableSorting: false,
    enableHiding: false,
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
          <TooltipContent className="bg-muted">
            <Time
              date={row.getValue('createdAt')}
              variant="timestamp"
              className="font-mono text-foreground"
            />
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    ),
    enableSorting: false,
    enableHiding: true,
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
                <Time date={startedAt} variant="timeSince" />
              </span>
            </TooltipTrigger>
            <TooltipContent className="bg-muted">
              <Time
                date={startedAt}
                variant="timestamp"
                asChild
                className="font-mono text-foreground"
              />
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    },
    enableSorting: false,
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
    accessorKey: 'additionalMetadata',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Metadata" />
    ),
    cell: ({ row }) => {
      return <AdditionalMetadataCell row={row} />;
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

export function AdditionalMetadataCell({ row }: { row: Row<V1TaskSummary> }) {
  const { setFilter, filters } = useFilters<RunsFilters>();

  const metadata = row.original.additionalMetadata;
  if (!metadata) {
    return <span>-</span>;
  }

  return (
    <AdditionalMetadata
      metadata={metadata}
      onClick={(click) => {
        setFilter('additional_metadata', [
          ...(filters.additional_metadata || []),
          `${click.key}:${click.value}`,
        ]);
      }}
    />
  );
}
