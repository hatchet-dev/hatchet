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
import { RunId } from '../run-id';
import { RunsBadge } from '../runs-badge';
import { Duration } from '@/next/components/ui/duration';
import { AdditionalMetadata } from '@/next/components/ui/additional-meta';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useRuns } from '@/next/hooks/use-runs';
import { Checkbox } from '@/next/components/ui/checkbox';
import { Button } from '@/next/components/ui/button';
import {
  ArrowDownFromLine,
  ChevronDownIcon,
  ChevronRightIcon,
} from 'lucide-react';
import { cn } from '@/next/lib/utils';

export const columns = (
  rowClicked?: (row: V1TaskSummary) => void,
  selectAll?: boolean,
  allowSelection: boolean = true,
): ColumnDef<V1TaskSummary>[] => {
  const selectCheckboxColumn: ColumnDef<V1TaskSummary> = {
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
      <div
        className={cn(
          `pl-${row.depth * 4}`,
          'flex flex-row items-center justify-start gap-x-2 max-w-6 mr-2',
        )}
      >
        <Checkbox
          checked={selectAll || row.getIsSelected()}
          onCheckedChange={(value: boolean) => row.toggleSelected(!!value)}
          aria-label="Select row"
          disabled={selectAll}
          onClick={(e) => e.stopPropagation()}
        />
        {row.getCanExpand() && (
          <Button
            onClick={(e) => {
              e.stopPropagation();
              row.toggleExpanded();
            }}
            variant="link"
            size="icon"
            className="cursor-pointer px-2"
            tooltip="Show tasks"
          >
            {row.getIsExpanded() ? (
              <ChevronDownIcon className="size-4" />
            ) : (
              <ChevronRightIcon className="size-4" />
            )}
          </Button>
        )}
      </div>
    ),
    enableSorting: false,
    enableHiding: false,
  };

  const contentColumns: ColumnDef<V1TaskSummary>[] = [
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="" />
      ),
      cell: ({ row }) => {
        const status = row.getValue('status') as V1TaskStatus;
        return (
          <div className="flex items-center justify-center h-full">
            <RunsBadge variant="xs" status={status} />
          </div>
        );
      },
      filterFn: (row, id, value: string) => {
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
        const url = ROUTES.runs.detail(
          row.original.tenantId,
          row.original.workflowRunExternalId || '',
        );
        return (
          <div className="flex items-center gap-2">
            <RunId
              taskRun={row.original}
              onClick={() => {
                rowClicked?.(row.original);
              }}
            />
            {url ? (
              <Link
                to={url}
                className="opacity-0 group-hover:opacity-100 transition-opacity text-muted-foreground hover:text-foreground"
                onClick={(e) => e.stopPropagation()}
              >
                <Button
                  variant="link"
                  tooltip="Drill down into run"
                  size="icon"
                >
                  <ArrowDownFromLine className="w-4 h-4" />
                </Button>
              </Link>
            ) : null}
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
        <Time
          date={row.getValue('createdAt')}
          variant="timeSince"
          tooltipVariant="timestamp"
        />
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
        const startedAt: string = row.getValue('startedAt');

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
        const startedAt: string | null = row.getValue('startedAt');
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

  if (allowSelection) {
    return [selectCheckboxColumn, ...contentColumns];
  }

  return contentColumns;
};

function AdditionalMetadataCell({ row }: { row: Row<V1TaskSummary> }) {
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
