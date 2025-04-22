'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import {
  V1TaskSummary,
  V1TaskStatus,
  WorkflowRunOrderByField,
  V1WorkflowType,
} from '@/lib/api';
import { Time } from '@/next/components/ui/time';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { DataTableColumnHeader } from './data-table-column-header';
import { RunId } from '../run-id';
import { RunsBadge } from '../runs-badge';
import { Duration } from '@/next/components/ui/duration';
import { AdditionalMetadata } from '@/next/components/ui/additional-meta';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useRuns } from '@/next/hooks/use-runs';
import { Checkbox } from '@/next/components/ui/checkbox';

export const statusOptions = [
  { label: 'Pending', value: 'PENDING' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
];

export const columns = (
  rowClicked?: (row: V1TaskSummary) => void,
  selectAll?: boolean,
): ColumnDef<V1TaskSummary>[] => [
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={selectAll || table.getIsAllPageRowsSelected()}
        onCheckedChange={(value: boolean) =>
          table.toggleAllPageRowsSelected(!!value)
        }
        aria-label="Select all"
        className="translate-y-[2px]"
        disabled={selectAll}
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={selectAll || row.getIsSelected()}
        onCheckedChange={(value: boolean) => row.toggleSelected(!!value)}
        aria-label="Select row"
        className="translate-y-[2px]"
        disabled={selectAll}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
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
    cell: ({ row }) => {
      const url =
        row.original.type === V1WorkflowType.TASK
          ? undefined
          : ROUTES.runs.detail(row.original.taskExternalId || '');

      return (
        <div className="flex items-center gap-2">
          <RunId
            taskRun={row.original}
            onClick={rowClicked ? () => rowClicked(row.original) : undefined}
          />
          {url && (
            <Link
              to={url}
              className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-foreground"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
                <polyline points="15 3 21 3 21 9" />
                <line x1="10" y1="14" x2="21" y2="3" />
              </svg>
            </Link>
          )}
        </div>
      );
    },
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
];

export function AdditionalMetadataCell({ row }: { row: Row<V1TaskSummary> }) {
  const {
    filters: { setFilter, filters },
  } = useRuns();

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
