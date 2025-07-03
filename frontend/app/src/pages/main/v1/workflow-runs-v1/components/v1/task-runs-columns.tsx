import { ColumnDef } from '@tanstack/react-table';
import { Link } from 'react-router-dom';
import {
  AdditionalMetadata,
  AdditionalMetadataClick,
} from '../../../events/components/additional-metadata';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { ChevronDownIcon, ChevronRightIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { V1RunStatus } from '../../../workflow-runs/components/run-statuses';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { Duration } from '@/components/v1/shared/duration';

export const TaskRunColumn = {
  taskName: 'task_name',
  status: 'status',
  workflow: 'Workflow',
  parentTaskExternalId: 'parentTaskExternalId',
  createdAt: 'Created at',
  startedAt: 'Started at',
  finishedAt: 'Finished at',
  duration: 'Duration',
  additionalMetadata: 'additionalMetadata',
} as const;

export const columns: (
  tenantId: string,
  selectedAdditionalMetaRunId: string | null,
  setSelectedAdditionalMetaRunId: (runId: string | null) => void,
  onAdditionalMetadataClick?: (click: AdditionalMetadataClick) => void,
  onTaskRunIdClick?: (taskRunId: string) => void,
) => ColumnDef<V1TaskSummary>[] = (
  tenantId,
  selectedAdditionalMetaRunId,
  setSelectedAdditionalMetaRunId,
  onAdditionalMetadataClick,
  onTaskRunIdClick,
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
            variant="ghost"
            className="cursor-pointer px-2"
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
  // {
  //   accessorKey: 'task_id',
  //   header: ({ column }) => (
  //     <DataTableColumnHeader column={column} title="Id" />
  //   ),
  //   cell: ({ row }) => (
  //     <div
  //       className="cursor-pointer hover:underline min-w-fit whitespace-nowrap items-center flex-row flex gap-x-1"
  //       onClick={() => {
  //         navigator.clipboard.writeText(row.original.taskId.toString());
  //       }}
  //     >
  //       {row.original.taskId}
  //       <ClipboardDocumentIcon className="size-4 ml-1" />
  //     </div>
  //   ),
  //   enableSorting: false,
  //   enableHiding: true,
  // },
  {
    accessorKey: TaskRunColumn.taskName,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Task" />
    ),
    cell: ({ row }) => {
      if (row.getCanExpand()) {
        return (
          <Link to={`/tenants/${tenantId}/runs/${row.original.metadata.id}`}>
            <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
              {row.original.displayName}
            </div>
          </Link>
        );
      } else {
        return (
          <div
            className="cursor-pointer hover:underline min-w-fit whitespace-nowrap"
            onClick={() =>
              row.original.metadata.id &&
              onTaskRunIdClick &&
              onTaskRunIdClick(row.original.metadata.id)
            }
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
    accessorKey: TaskRunColumn.status,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => (
      <V1RunStatus
        status={row.original.status}
        errorMessage={row.original.errorMessage}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: TaskRunColumn.workflow,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Workflow" />
    ),
    cell: ({ row }) => {
      const workflowId = row.original.workflowId;
      const workflowName = row.original.workflowName;

      return (
        <div className="min-w-fit whitespace-nowrap">
          {(workflowId && workflowName && (
            <a href={`/tenants/${tenantId}/tasks/${workflowId}`}>
              {workflowName}
            </a>
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
    accessorKey: TaskRunColumn.parentTaskExternalId,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Parent Task External ID" />
    ),
    cell: () => null,
    enableSorting: false,
    enableHiding: true,
  },
  // {
  //   accessorKey: 'Triggered by',
  //   header: ({ column }) => (
  //     <DataTableColumnHeader column={column} title="Triggered by" />
  //   ),
  //   cell: ({ row }) => {
  //     return <div>{row.original.triggeredBy}</div>;
  //   },
  //   enableSorting: false,
  //   enableHiding: true,
  // },
  {
    accessorKey: TaskRunColumn.createdAt,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Created at"
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
    accessorKey: TaskRunColumn.startedAt,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Started at"
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
    accessorKey: TaskRunColumn.finishedAt,
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Finished at"
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
    accessorKey: TaskRunColumn.duration,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Duration" />
    ),
    cell: ({ row }) => {
      const startedAt = row.original.startedAt;
      const finishedAt = row.original.finishedAt;
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
    accessorKey: TaskRunColumn.additionalMetadata,
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
          onClick={onAdditionalMetadataClick}
          isOpen={selectedAdditionalMetaRunId === row.original.metadata.id}
          onOpenChange={(open) => {
            if (open) {
              setSelectedAdditionalMetaRunId(row.original.metadata.id);
            } else {
              setSelectedAdditionalMetaRunId(null);
            }
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
        <DataTableRowActions
          row={row}
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
