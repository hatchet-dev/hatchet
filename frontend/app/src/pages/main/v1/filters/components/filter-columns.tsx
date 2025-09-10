import { ColumnDef } from '@tanstack/react-table';
import { V1Filter } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { CheckIcon } from 'lucide-react';

export const FilterColumn = {
  id: 'ID',
  workflowId: 'Workflow',
  scope: 'Scope',
  expression: 'Expression',
  isDeclarative: 'Is Declarative',
} as const;

export type FilterColumnKeys = keyof typeof FilterColumn;

export const idKey = 'id';
export const workflowIdKey: FilterColumnKeys = 'workflowId';
export const scopeKey: FilterColumnKeys = 'scope';
export const expressionKey: FilterColumnKeys = 'expression';
export const isDeclarativeKey: FilterColumnKeys = 'isDeclarative';

export const filterColumns = (
  workflowIdToName: Record<string, string>,
  onRowClick: (filter: V1Filter) => void,
): ColumnDef<V1Filter>[] => {
  return [
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={FilterColumn.id} />
      ),
      cell: ({ row }) => (
        <div
          className="w-full cursor-pointer hover:text-blue-600 transition-colors"
          onClick={() => onRowClick(row.original)}
        >
          {row.original.metadata.id}
        </div>
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
    {
      accessorKey: isDeclarativeKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={FilterColumn.isDeclarative}
        />
      ),
      cell: ({ row }) =>
        row.original.isDeclarative ? (
          <CheckIcon className="h-4 w-4 text-green-600" />
        ) : null,
      enableSorting: false,
      enableHiding: true,
    },
  ];
};
