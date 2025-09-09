import { ColumnDef } from '@tanstack/react-table';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { V1Filter } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';

export const FilterColumn = {
  id: 'Id',
  workflowId: 'Workflow',
  scope: 'Scope',
  expression: 'Expression',
} as const;

export type FilterColumnKeys = keyof typeof FilterColumn;

export const idKey = 'id';
export const workflowIdKey: FilterColumnKeys = 'workflowId';
export const scopeKey: FilterColumnKeys = 'scope';
export const expressionKey: FilterColumnKeys = 'expression';

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
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={FilterColumn.id} />
      ),
      cell: ({ row }) => (
        <div className="w-full">{row.original.metadata.id}</div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: workflowIdKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={FilterColumn.workflowId}
        />
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
      accessorKey: scopeKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={FilterColumn.scope} />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.scope}</div>,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: expressionKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={FilterColumn.expression}
        />
      ),
      cell: ({ row }) => (
        <CodeHighlighter
          language="text"
          className="whitespace-pre-wrap break-words text-sm leading-relaxed"
          code={row.original.expression}
          copy={false}
        />
      ),
      enableSorting: false,
      enableHiding: true,
    },
  ];
};
