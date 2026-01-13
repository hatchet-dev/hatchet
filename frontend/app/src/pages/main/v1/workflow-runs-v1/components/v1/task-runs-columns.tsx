import {
  AdditionalMetadata,
  AdditionalMetadataClick,
} from '../../../events/components/additional-metadata';
import { V1RunStatus } from '../../../workflow-runs/components/run-statuses';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Duration } from '@/components/v1/shared/duration';
import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { ChevronDownIcon, ChevronRightIcon } from '@heroicons/react/24/outline';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const TaskRunColumn = {
  taskName: 'Task Name',
  status: 'Status',
  workflow: 'Workflow',
  parentTaskExternalId: 'Parent Task External ID',
  flattenDAGs: 'Flatten DAGs',
  createdAt: 'Created At',
  startedAt: 'Started At',
  finishedAt: 'Finished At',
  duration: 'Duration',
  additionalMetadata: 'Metadata',
} as const;

export type TaskRunColumnKeys = keyof typeof TaskRunColumn;

export const workflowKey: TaskRunColumnKeys = 'workflow';
const parentTaskExternalIdKey: TaskRunColumnKeys = 'parentTaskExternalId';
export const flattenDAGsKey: TaskRunColumnKeys = 'flattenDAGs';
export const createdAtKey: TaskRunColumnKeys = 'createdAt';
const startedAtKey: TaskRunColumnKeys = 'startedAt';
const finishedAtKey: TaskRunColumnKeys = 'finishedAt';
const durationKey: TaskRunColumnKeys = 'duration';
export const additionalMetadataKey: TaskRunColumnKeys = 'additionalMetadata';
const taskNameKey: TaskRunColumnKeys = 'taskName';
export const statusKey: TaskRunColumnKeys = 'status';

export const createdAfterKey = 'createdAfter';
export const finishedBeforeKey = 'finishedBefore';
export const isCustomTimeRangeKey = 'isCustomTimeRange';
export const timeWindowKey = 'timeWindow';

export const columns: (
  tenantId: string,
  selectedAdditionalMetaRunId: string | null,
  onAdditionalMetadataClick: (click: AdditionalMetadataClick) => void,
  onTaskRunIdClick: (taskRunId: string) => void,
  onAdditionalMetadataOpenChange: (rowId: string, open: boolean) => void,
) => ColumnDef<V1TaskSummary>[] = (
  tenantId,
  selectedAdditionalMetaRunId,
  onAdditionalMetadataClick,
  onTaskRunIdClick,
  onAdditionalMetadataOpenChange,
) => [
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
      <div
        className={cn(
          `pl-${row.depth * 4}`,
          'flex flex-row items-center justify-start gap-x-2',
        )}
      >
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
        {row.getCanExpand() && (
          <Button
            onClick={() => row.toggleExpanded()}
            variant="icon"
            className="px-2"
            hoverText="Show tasks"
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
  },
  {
    accessorKey: taskNameKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={TaskRunColumn.taskName} />
    ),
    cell: ({ row }) => {
      if (row.getCanExpand()) {
        return (
          <Link
            to={appRoutes.tenantRunRoute.to}
            params={{ tenant: tenantId, run: row.original.metadata.id }}
          >
            <div className="min-w-fit cursor-pointer whitespace-nowrap hover:underline">
              {row.original.displayName}
            </div>
          </Link>
        );
      } else {
        return (
          <div
            className="min-w-fit cursor-pointer whitespace-nowrap hover:underline"
            onClick={() => onTaskRunIdClick(row.original.metadata.id)}
          >
            {row.original.displayName}
          </div>
        );
      }
    },
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: statusKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={TaskRunColumn.status} />
    ),
    cell: ({ row }) => (
      <V1RunStatus
        className="items-center justify-center px-2 text-center"
        status={row.original.status}
        errorMessage={row.original.errorMessage}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: workflowKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={TaskRunColumn.workflow} />
    ),
    cell: ({ row }) => {
      const workflowId = row.original.workflowId;
      const workflowName = row.original.workflowName;

      return (
        <div className="min-w-fit whitespace-nowrap">
          {(workflowId && workflowName && (
            <Link
              to={appRoutes.tenantWorkflowRoute.to}
              params={{ tenant: tenantId, workflow: workflowId }}
            >
              {workflowName}
            </Link>
          )) ||
            'N/A'}
        </div>
      );
    },
    show: false,
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: parentTaskExternalIdKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.parentTaskExternalId}
      />
    ),
    cell: () => null,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: flattenDAGsKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.flattenDAGs}
      />
    ),
    cell: () => null,
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: createdAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.createdAt}
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          {row.original.metadata.createdAt ? (
            <RelativeDate date={row.original.metadata.createdAt} />
          ) : (
            'N/A'
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: startedAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.startedAt}
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          {row.original.startedAt ? (
            <RelativeDate date={row.original.startedAt} />
          ) : (
            'N/A'
          )}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: finishedAtKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.finishedAt}
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      const finishedAt = row.original.finishedAt ? (
        <RelativeDate date={row.original.finishedAt} />
      ) : (
        'N/A'
      );

      return <div className="whitespace-nowrap">{finishedAt}</div>;
    },
    enableSorting: false,
    enableHiding: true,
  },
  {
    accessorKey: durationKey,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={TaskRunColumn.duration} />
    ),
    cell: ({ row }) => {
      const startedAt = row.original.startedAt;
      const finishedAt = row.original.finishedAt;
      const status = row.getValue(statusKey) as V1TaskStatus;

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
    accessorKey: additionalMetadataKey,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={TaskRunColumn.additionalMetadata}
      />
    ),
    cell: ({ row }) => {
      if (!row.original.additionalMetadata) {
        return <div></div>;
      }

      return (
        <AdditionalMetadata
          metadata={row.original.additionalMetadata}
          onClick={onAdditionalMetadataClick}
          isOpen={selectedAdditionalMetaRunId === row.original.metadata.id}
          onOpenChange={(open) => {
            onAdditionalMetadataOpenChange(row.original.metadata.id, open);
          }}
        />
      );
    },
    enableSorting: false,
  },
  {
    id: 'actions',
    cell: ({ row }) => {
      return (
        <TableRowActions
          row={row.original}
          actions={[
            {
              label: 'Copy Run Id',
              onClick: () => {
                navigator.clipboard.writeText(row.original.metadata.id);
              },
            },
          ]}
        />
      );
    },
  },
];
