import { ColumnDef } from '@tanstack/react-table';
import { V1Filter } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Button } from '@/components/v1/ui/button';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import {
  CheckIcon,
  CopyIcon,
  EyeIcon,
  Trash2Icon,
  TrashIcon,
} from 'lucide-react';
import { FilterPayloadPopover } from '../../events/components/filter-payload-popover';
import { useState } from 'react';

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
  handleDelete: (filterId: string) => Promise<void>,
): ColumnDef<V1Filter>[] => {
  return [
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
    {
      accessorKey: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="" />
      ),
      cell: ({ row }) => {
        const filter = row.original;
        const payload = row.original.payload;
        const payloadString = JSON.stringify(payload, null, 2);
        // eslint-disable-next-line react-hooks/rules-of-hooks
        const [copiedItem, setCopiedItem] = useState<string | null>(null);
        // eslint-disable-next-line react-hooks/rules-of-hooks
        const [isDropdownOpen, setIsDropdownOpen] = useState(false);
        // eslint-disable-next-line react-hooks/rules-of-hooks
        const [isPayloadPopoverOpen, setIsPayloadPopoverOpen] = useState(false);

        const handleCopy = (text: string, label: string) => {
          navigator.clipboard.writeText(text);
          setCopiedItem(label);
          setTimeout(() => setCopiedItem(null), 1200);
          setTimeout(() => setIsDropdownOpen(false), 300);
        };

        const handleViewPayload = (e: React.MouseEvent) => {
          e.preventDefault();
          e.stopPropagation();
          setIsDropdownOpen(false);
          setTimeout(() => setIsPayloadPopoverOpen(true), 100);
        };

        return (
          <div className="flex justify-center">
            <DropdownMenu
              open={isDropdownOpen}
              onOpenChange={setIsDropdownOpen}
            >
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="h-8 w-8 p-0 hover:bg-muted/50"
                >
                  <DotsVerticalIcon className="h-4 w-4 text-muted-foreground cursor-pointer" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleCopy(filter.metadata.id, 'filter');
                  }}
                  className="flex items-center gap-2 cursor-pointer"
                >
                  {copiedItem === 'filter' ? (
                    <CheckIcon className="h-4 w-4 text-green-600" />
                  ) : (
                    <CopyIcon className="h-4 w-4" />
                  )}
                  {copiedItem === 'filter'
                    ? 'Copied Filter ID!'
                    : 'Copy Filter ID'}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={handleViewPayload}
                  className="flex items-center gap-2 cursor-pointer"
                >
                  <EyeIcon className="h-4 w-4" />
                  View Payload
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleDelete(filter.metadata.id)}
                  className="flex items-center gap-2 cursor-pointer"
                >
                  <Trash2Icon className="size-4" />
                  Delete Filter
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <FilterPayloadPopover
              isOpen={isPayloadPopoverOpen}
              setIsOpen={setIsPayloadPopoverOpen}
              content={payloadString}
            />
          </div>
        );
      },
      enableSorting: false,
      enableHiding: false,
      size: 50,
      minSize: 50,
      maxSize: 50,
    },
  ];
};
