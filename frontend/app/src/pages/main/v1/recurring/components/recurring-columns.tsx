import { AdditionalMetadata } from '../../events/components/additional-metadata';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { TableRowActions } from '@/components/v1/molecules/data-table/data-table-row-actions';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Spinner } from '@/components/v1/ui/loading';
import { CronWorkflows } from '@/lib/api';
import { extractCronTz, formatCron } from '@/lib/cron';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { ColumnDef } from '@tanstack/react-table';
import { Check, X } from 'lucide-react';

export const CronColumn = {
  expression: 'Expression',
  description: 'Description',
  timezone: 'Timezone',
  name: 'Name',
  workflow: 'Workflow',
  metadata: 'Metadata',
  createdAt: 'Created At',
  actions: 'Actions',
  enabled: 'Enabled',
};

type CronColumnKeys = keyof typeof CronColumn;

const enabledKey: CronColumnKeys = 'enabled';
const expressionKey: CronColumnKeys = 'expression';
const descriptionKey: CronColumnKeys = 'description';
const timezoneKey: CronColumnKeys = 'timezone';
const nameKey: CronColumnKeys = 'name';
export const workflowKey: CronColumnKeys = 'workflow';
export const metadataKey: CronColumnKeys = 'metadata';
const createdAtKey: CronColumnKeys = 'createdAt';
const actionsKey: CronColumnKeys = 'actions';

export const columns = ({
  tenantId,
  onDeleteClick,
  onEnableClick,
  selectedJobId,
  setSelectedJobId,
  isUpdatePending,
  updatingCronId,
}: {
  tenantId: string;
  onDeleteClick: (row: CronWorkflows) => void;
  onEnableClick: (row: CronWorkflows) => void;
  selectedJobId: string | null;
  setSelectedJobId: (jobId: string | null) => void;
  isUpdatePending: boolean;
  updatingCronId: string | undefined;
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
    {
      accessorKey: enabledKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.enabled} />
      ),
      cell: ({ row }) => (
        <div>
          {isUpdatePending && updatingCronId === row.original.metadata.id ? (
            <Spinner />
          ) : row.original.enabled ? (
            <Check className="size-4 text-emerald-500" />
          ) : (
            <X className="size-4 text-red-500" />
          )}
        </div>
      ),
    },
    {
      accessorKey: actionsKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={CronColumn.actions} />
      ),
      cell: ({ row }) => (
        <div className="flex flex-row justify-center">
          <TableRowActions
            row={row.original}
            actions={[
              {
                label: 'Delete',
                onClick: () => onDeleteClick(row.original),
                disabled:
                  row.original.method !== 'API'
                    ? 'This cron was created via a code definition. Delete it from the code definition instead.'
                    : undefined,
              },
              {
                label: row.original.enabled ? 'Disable' : 'Enable',
                onClick: () => onEnableClick(row.original),
                disabled:
                  isUpdatePending && updatingCronId === row.original.metadata.id
                    ? 'Update in progress'
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
