import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../../components/molecules/data-table/data-table-column-header';
import { Link } from 'react-router-dom';
import {
  AdditionalMetadata,
  AdditionalMetadataClick,
} from '../../../events/components/additional-metadata';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { ListableWorkflowRun } from '../workflow-runs-table';
import { ChevronDownIcon, ChevronRightIcon } from '@heroicons/react/24/outline';
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { V2RunStatus } from '@/pages/main/workflow-runs/components/run-statuses';

export const columns: (
  onAdditionalMetadataClick?: (click: AdditionalMetadataClick) => void,
) => ColumnDef<ListableWorkflowRun>[] = (onAdditionalMetadataClick) => [
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
    accessorKey: 'task_name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Task" />
    ),
    cell: ({ row }) => (
      <Link to={'/workflow-runs/' + row.original.metadata.id}>
        <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
          {row.original.displayName}
        </div>
      </Link>
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => (
      <V2RunStatus
        status={row.original.status}
        errorMessage={row.original.errorMessage}
      />
    ),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'Workflow',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Workflow" />
    ),
    cell: ({ row }) => {
      const workflowId = row.original?.workflowId;
      const workflowName = row.original.workflowName;

      return (
        <div className="min-w-fit whitespace-nowrap">
          {(workflowId && workflowName && (
            <a href={`/workflows/${workflowId}`}>{workflowName}</a>
          )) ||
            'N/A'}
        </div>
      );
    },
    show: false,
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
    accessorKey: 'Created at',
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
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'Started at',
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
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'Finished at',
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
    enableSorting: true,
    enableHiding: true,
  },
  {
    accessorKey: 'Duration',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Duration (ms)"
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return <div className="whitespace-nowrap">{row.original.duration}</div>;
    },
    enableSorting: true,
    enableHiding: true,
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
          onClick={onAdditionalMetadataClick}
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
