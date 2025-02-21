import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { RateLimit, ScheduledWorkflows } from '@/lib/api';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { RunStatus } from '../../workflow-runs/components/run-statuses';
import { Link } from 'react-router-dom';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
export type RateLimitRow = RateLimit & {
  metadata: {
    id: string;
  };
};

export const columns = ({
  onDeleteClick,
}: {
  onDeleteClick: (row: ScheduledWorkflows) => void;
}): ColumnDef<ScheduledWorkflows>[] => {
  return [
    {
      accessorKey: 'runId',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Run ID" />
      ),
      cell: ({ row }) => {
        return row.original.workflowRunId ? (
          <Link to={`/workflow-runs/${row.original.workflowRunId}`}>
            <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
              {row.original.workflowRunName}
            </div>
          </Link>
        ) : null;
      },
    },
    {
      accessorKey: 'status',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Status" />
      ),
      cell: ({ row }) => (
        <RunStatus status={row.original.workflowRunStatus || 'SCHEDULED'} />
      ),
    },
    {
      accessorKey: 'triggerAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Trigger At" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <RelativeDate date={row.original.triggerAt} />
        </div>
      ),
    },
    {
      accessorKey: 'Workflow',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Workflow" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
            <a href={`/workflows/${row.original.workflowId}`}>
              {row.original.workflowName}
            </a>
          </div>
        </div>
      ),
      enableSorting: false,
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
          <AdditionalMetadata metadata={row.original.additionalMetadata} />
        );
      },
      enableSorting: false,
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Created At" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
      enableSorting: true,
      enableHiding: true,
    },
    {
      accessorKey: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Actions" />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row justify-center">
          <DataTableRowActions
            row={row}
            actions={[
              {
                label: 'Delete',
                onClick: () => onDeleteClick(row.original),
                disabled:
                  row.original.method !== 'API'
                    ? 'Cannot delete scheduled workflow created via workflow code definition'
                    : undefined,
              },
            ]}
          />
        </div>
      ),
      enableHiding: true,
      enableSorting: false,
    },
  ];
};
