import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '../../../../components/molecules/data-table/data-table-column-header';
import { RateLimit, ScheduledWorkflows } from '@/lib/api';
import RelativeDate from '@/components/molecules/relative-date';
import { AdditionalMetadata } from '../../events/components/additional-metadata';

export type RateLimitRow = RateLimit & {
  metadata: {
    id: string;
  };
};

export const columns: ColumnDef<ScheduledWorkflows>[] = [
  {
    accessorKey: 'triggerAt',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Trigger At" />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center gap-4 pl-4">
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
      <div className="flex flex-row items-center gap-4 pl-4">
        <a href={`/workflows/${row.original.workflowId}`}>
          {row.original.workflowName}
        </a>
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

      return <AdditionalMetadata metadata={row.original.additionalMetadata} />;
    },
    enableSorting: false,
  },
  {
    accessorKey: 'createdAt',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Created At" />
    ),
    cell: ({ row }) => (
      <div className="flex flex-row items-center gap-4 pl-4">
        <RelativeDate date={row.original.metadata.createdAt} />
      </div>
    ),
    enableSorting: true,
    enableHiding: true,
  },
];
