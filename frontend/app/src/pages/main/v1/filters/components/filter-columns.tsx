import { ColumnDef } from '@tanstack/react-table';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { V1Filter } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';

export const columns = (
  workflowIdToName: Record<string, string>,
): ColumnDef<V1Filter>[] => {
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
          className="translate-y-[2px]"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    },
    {
      accessorKey: 'id',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Id" />
      ),
      cell: ({ row }) => (
        <div className="w-full">{row.original.metadata.id}</div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'workflowId',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Workflow" />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          {workflowIdToName[row.original.workflowId]}
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'scope',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Scope" />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.scope}</div>,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'expression',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Expression" />
      ),
      cell: ({ row }) => (
        <div className="w-full">{row.original.expression}</div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
  ];
};
