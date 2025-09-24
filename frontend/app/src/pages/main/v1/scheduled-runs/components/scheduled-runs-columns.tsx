import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/molecules/data-table/data-table-column-header';
import { ScheduledWorkflows } from '@/lib/api';
import RelativeDate from '@/components/molecules/relative-date';
import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { RunStatus } from '../../workflow-runs/components/run-statuses';
import { DataTableRowActions } from '@/components/molecules/data-table/data-table-row-actions';
import { Link } from 'react-router-dom';

export const ScheduledRunColumn = {
  runId: 'Run ID',
  status: 'Status',
  triggerAt: 'Trigger At',
  workflow: 'Workflow',
  metadata: 'Metadata',
  createdAt: 'Created At',
  actions: 'Actions',
};

export type ScheduledRunColumnKeys = keyof typeof ScheduledRunColumn;

export const idKey: ScheduledRunColumnKeys = 'runId';
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
}: {
  tenantId: string;
  onDeleteClick: (row: ScheduledWorkflows) => void;
  selectedAdditionalMetaJobId: string | null;
  handleSetSelectedAdditionalMetaJobId: (runId: string | null) => void;
}): ColumnDef<ScheduledWorkflows>[] => {
  return [
    {
      accessorKey: idKey,
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
        <RunStatus status={row.original.workflowRunStatus || 'SCHEDULED'} />
      ),
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
        <div className="flex flex-row items-center gap-4">
          <RelativeDate date={row.original.triggerAt} />
        </div>
      ),
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
      enableSorting: true,
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
      enableHiding: true,
      enableSorting: false,
    },
  ];
};
