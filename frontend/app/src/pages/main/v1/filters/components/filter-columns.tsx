import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { V1Filter } from '@/lib/api';
import { ColumnDef } from '@tanstack/react-table';
import { CheckIcon } from 'lucide-react';

export const FilterColumn = {
  id: 'ID',
  workflowId: 'Workflow',
  scope: 'Scope',
  expression: 'Expression',
  isDeclarative: 'Is Declarative',
} as const;

type FilterColumnKeys = keyof typeof FilterColumn;

const idKey = 'id';
export const workflowIdKey: FilterColumnKeys = 'workflowId';
export const scopeKey: FilterColumnKeys = 'scope';
const expressionKey: FilterColumnKeys = 'expression';
export const isDeclarativeKey: FilterColumnKeys = 'isDeclarative';

export const filterColumns = (
  workflowIdToName: Record<string, string>,
  onRowClick?: (filter: V1Filter) => void,
): ColumnDef<V1Filter>[] => {
  return [
    {
      accessorKey: idKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={FilterColumn.id} />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <Button
            className="w-fit pl-0"
            variant="link"
            onClick={() => {
              onRowClick?.(row.original);
            }}
          >
            {row.original.metadata.id}
          </Button>
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
          maxHeight="10rem"
          minWidth="20rem"
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
          <CheckIcon className="size-4 text-green-600" />
        ) : null,
      enableSorting: false,
      enableHiding: true,
    },
  ];
};
