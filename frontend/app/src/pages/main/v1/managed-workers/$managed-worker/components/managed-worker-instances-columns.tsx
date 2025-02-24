import { ColumnDef } from '@tanstack/react-table';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Instance } from '@/lib/api/generated/cloud/data-contracts';

export type InstanceWithMetadata = Instance & {
  metadata: {
    id: string;
  };
};

export const columns: ColumnDef<InstanceWithMetadata>[] = [
  {
    accessorKey: 'name',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => (
      <div className="cursor-pointer hover:underline min-w-fit whitespace-nowrap text-md p-2">
        {row.original.name}
      </div>
    ),
    enableSorting: true,
    enableHiding: false,
  },
  {
    accessorKey: 'state',
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="State" />
    ),
    cell: ({ row }) => (
      <div className="whitespace-nowrap">{row.original.state}</div>
    ),
    enableSorting: true,
    enableHiding: false,
  },
  {
    accessorKey: 'commitSha',
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title="Commit"
        className="whitespace-nowrap"
      />
    ),
    cell: ({ row }) => {
      return (
        <div className="whitespace-nowrap">
          {row.original.commitSha.substring(0, 7)}
        </div>
      );
    },
    enableSorting: false,
    enableHiding: false,
  },
];
