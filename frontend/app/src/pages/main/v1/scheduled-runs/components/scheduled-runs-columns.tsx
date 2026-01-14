import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { RunStatus } from '../../workflow-runs/components/run-statuses';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { ScheduledWorkflows } from '@/lib/api';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';

export const ScheduledRunColumn = {
  id: 'ID',
  status: 'Status',
  triggerAt: 'Trigger At',
  workflow: 'Workflow',
  metadata: 'Metadata',
  createdAt: 'Created At',
  actions: 'Actions',
};

type ScheduledRunColumnKeys = keyof typeof ScheduledRunColumn;

const idKey: ScheduledRunColumnKeys = 'id';
export const statusKey: ScheduledRunColumnKeys = 'status';
const triggerAtKey: ScheduledRunColumnKeys = 'triggerAt';
export const workflowKey: ScheduledRunColumnKeys = 'workflow';
export const metadataKey: ScheduledRunColumnKeys = 'metadata';
const createdAtKey: ScheduledRunColumnKeys = 'createdAt';
const actionsKey: ScheduledRunColumnKeys = 'actions';

export const columns = ({
  tenantId,
  onDeleteClick,
  onRescheduleClick,
  selectedAdditionalMetaJobId,
  handleSetSelectedAdditionalMetaJobId,
  onRowClick,
}: {
  tenantId: string;
  onDeleteClick: (row: ScheduledWorkflows) => void;
  onRescheduleClick: (row: ScheduledWorkflows) => void;
  selectedAdditionalMetaJobId: string | null;
  handleSetSelectedAdditionalMetaJobId: (runId: string | null) => void;
  onRowClick?: (row: ScheduledWorkflows) => void;
}): ColumnDef<ScheduledWorkflows>[] => {
  return [
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
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
          disabled={row.original.method !== 'API' ? true : undefined}
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={ScheduledRunColumn.id} />
      ),
      cell: ({ row }) => (
        <div
          className="min-w-fit cursor-pointer whitespace-nowrap hover:underline"
          onClick={() => onRowClick?.(row.original)}
        >
          {row.original.metadata.id}
        </div>
      ),
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
          className="flex cursor-pointer flex-row items-center gap-4"
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
          <div className="min-w-fit cursor-pointer whitespace-nowrap hover:underline">
            <Link
              to={appRoutes.tenantWorkflowRoute.to}
              params={{ tenant: tenantId, workflow: row.original.workflowId }}
            >
              {row.original.workflowName}
            </Link>
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
          <TableRowActions
            row={row.original}
            actions={[
              {
                label: 'Reschedule',
                onClick: () => onRescheduleClick(row.original),
                disabled:
                  row.original.method !== 'API'
                    ? 'Cannot reschedule scheduled run created via code definition'
                    : row.original.workflowRunId
                      ? 'Cannot reschedule a scheduled run that has already been triggered'
                      : undefined,
              },
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
