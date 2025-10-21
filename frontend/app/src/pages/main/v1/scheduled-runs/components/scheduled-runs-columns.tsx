import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { RateLimit, ScheduledWorkflows } from '@/lib/api';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { RunStatus } from '../../workflow-runs/components/run-statuses';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { Link } from 'react-router-dom';

export type RateLimitRow = RateLimit & {
  metadata: {
    id: string;
  };
};

export const ScheduledRunColumn = {
  id: 'ID',
  runId: 'Run ID',
  status: 'Status',
  triggerAt: 'Trigger At',
  workflow: 'Workflow',
  metadata: 'Metadata',
  createdAt: 'Created At',
  actions: 'Actions',
};

export type ScheduledRunColumnKeys = keyof typeof ScheduledRunColumn;

export const idKey: ScheduledRunColumnKeys = 'id';
export const runIdKey: ScheduledRunColumnKeys = 'runId';
export const statusKey: ScheduledRunColumnKeys = 'status';
export const triggerAtKey: ScheduledRunColumnKeys = 'triggerAt';
export const workflowKey: ScheduledRunColumnKeys = 'workflow';
export const metadataKey: ScheduledRunColumnKeys = 'metadata';
export const createdAtKey: ScheduledRunColumnKeys = 'createdAt';
export const actionsKey: ScheduledRunColumnKeys = 'actions';

export const columns = ({
  tenantId,
  onDeleteClick,
  selectedAdditionalMetaJobId,
  handleSetSelectedAdditionalMetaJobId,
  onRowClick,
}: {
  tenantId: string;
  onDeleteClick: (row: ScheduledWorkflows) => void;
  selectedAdditionalMetaJobId: string | null;
  handleSetSelectedAdditionalMetaJobId: (runId: string | null) => void;
  onRowClick?: (row: ScheduledWorkflows) => void;
}): ColumnDef<ScheduledWorkflows>[] => {
  return [
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={ScheduledRunColumn.id} />
      ),
      cell: ({ row }) => (
        <div
          className="cursor-pointer hover:underline min-w-fit whitespace-nowrap"
          onClick={() => onRowClick?.(row.original)}
        >
          {row.original.metadata.id}
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: runIdKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.runId}
        />
      ),
      cell: ({ row }) =>
        row.original.workflowRunId ? (
          <Link to={`/tenants/${tenantId}/runs/${row.original.workflowRunId}`}>
            <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
              {row.original.workflowRunName}
            </div>
          </Link>
        ) : null,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: statusKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.status}
        />
      ),
      cell: ({ row }) => (
        <div
          className="cursor-pointer"
          onClick={() => onRowClick?.(row.original)}
        >
          <RunStatus status={row.original.workflowRunStatus || 'SCHEDULED'} />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: triggerAtKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.triggerAt}
        />
      ),
      cell: ({ row }) => (
        <div
          className="flex flex-row items-center gap-4 cursor-pointer"
          onClick={() => onRowClick?.(row.original)}
        >
          <RelativeDate date={row.original.triggerAt} />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: workflowKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.workflow}
        />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
            <a
              href={`/tenants/${tenantId}/workflows/${row.original.workflowId}`}
            >
              {row.original.workflowName}
            </a>
          </div>
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: metadataKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.metadata}
        />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.additionalMetadata}
            isOpen={selectedAdditionalMetaJobId === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                handleSetSelectedAdditionalMetaJobId(row.original.metadata.id);
              } else {
                handleSetSelectedAdditionalMetaJobId(null);
              }
            }}
          />
        );
      },
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: createdAtKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.createdAt}
        />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: actionsKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={ScheduledRunColumn.actions}
        />
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
                    ? 'Cannot delete scheduled run created via code definition'
                    : undefined,
              },
            ]}
          />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
  ];
};
