import { ColumnDef } from '@tanstack/react-table';
import { CronWorkflows } from '@/lib/api';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Link } from 'react-router-dom';
import { DataTableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { Badge } from '@/components/v1/ui/badge';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { extractCronTz, formatCron } from '@/lib/utils';

export const CronColumn = {
  expression: 'Expression',
  description: 'Description',
  timezone: 'Timezone',
  name: 'Name',
  workflow: 'Workflow',
  metadata: 'Metadata',
  createdAt: 'Created At',
  actions: 'Actions',
};

export type CronColumnKeys = keyof typeof CronColumn;

export const expressionKey: CronColumnKeys = 'expression';
export const descriptionKey: CronColumnKeys = 'description';
export const timezoneKey: CronColumnKeys = 'timezone';
export const nameKey: CronColumnKeys = 'name';
export const workflowKey: CronColumnKeys = 'workflow';
export const metadataKey: CronColumnKeys = 'metadata';
export const createdAtKey: CronColumnKeys = 'createdAt';
export const actionsKey: CronColumnKeys = 'actions';

export const columns = ({
  tenantId,
  onDeleteClick,
  selectedJobId,
  setSelectedJobId,
}: {
  tenantId: string;
  onDeleteClick: (row: CronWorkflows) => void;
  selectedJobId: string | null;
  setSelectedJobId: (jobId: string | null) => void;
}): ColumnDef<CronWorkflows>[] => {
  return [
    {
      accessorKey: expressionKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.expression} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4 whitespace-nowrap">
          {row.original.cron}
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: descriptionKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.description} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          Runs {formatCron(row.original.cron)}
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: timezoneKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.timezone} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          {extractCronTz(row.original.cron)}
        </div>
      ),
      enableSorting: false,
    },
    {
      accessorKey: nameKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.name} />
      ),
      cell: ({ row }) => (
        <div>
          {row.original.method === 'API' ? (
            row.original.name
          ) : (
            <Badge variant="outline">Defined in code</Badge>
          )}
        </div>
      ),
    },
    {
      accessorKey: workflowKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.workflow} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap">
            <Link
              to={`/tenants/${tenantId}/workflows/${row.original.workflowId}`}
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
        <DataTableColumnHeader column={column} title={CronColumn.metadata} />
      ),
      cell: ({ row }) => {
        if (!row.original.additionalMetadata) {
          return <div></div>;
        }

        return (
          <AdditionalMetadata
            metadata={row.original.additionalMetadata}
            isOpen={selectedJobId === row.original.metadata.id}
            onOpenChange={(open) => {
              if (open) {
                setSelectedJobId(row.original.metadata.id);
              } else {
                setSelectedJobId(null);
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
        <DataTableColumnHeader column={column} title={CronColumn.createdAt} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row items-center gap-4">
          <RelativeDate date={row.original.metadata.createdAt} />
        </div>
      ),
      enableSorting: true,
      enableHiding: true,
    },
    // {
    //   accessorKey: 'method',
    //   header: ({ column }) => (
    //     <DataTableColumnHeader column={column} title="Create Method" />
    //   ),
    //   cell: ({ row }) => <div>{row.original.method}</div>,
    // },
    // {
    //   accessorKey: 'enabled',
    //   header: ({ column }) => (
    //     <DataTableColumnHeader column={column} title="Enabled" />
    //   ),
    //   cell: ({ row }) => <div>{row.original.enabled ? 'Yes' : 'No'}</div>,
    // },
    {
      accessorKey: actionsKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.actions} />
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
                    ? 'This cron was created via a code definition. Delete it from the code definition instead.'
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
