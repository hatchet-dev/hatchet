import { ColumnDef, Row } from '@tanstack/react-table';
import { V1Webhook } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { Check, Copy, Loader, Trash2 } from 'lucide-react';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useState } from 'react';
import { useWebhooks } from '../hooks/use-webhooks';
import { SourceName } from './source-name';
import { AuthMethod } from './auth-method';

export const columns = (): ColumnDef<V1Webhook>[] => {
  return [
    {
      accessorKey: 'name',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Name" />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.name}</div>,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'sourceName',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Source" />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <SourceName sourceName={row.original.sourceName} />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'expression',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Expression" />
      ),
      cell: ({ row }) => (
        <code className="bg-muted relative rounded px-2 py-1 font-mono text-xs h-full">
          {row.original.eventKeyExpression}
        </code>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'authType',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Auth Method" />
      ),
      cell: ({ row }) => (
        <div className="w-full">
          <AuthMethod authMethod={row.original.authType} />
        </div>
      ),
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: 'actions',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="" />
      ),
      cell: ({ row }) => <WebhookActionsCell row={row} />,
      enableSorting: false,
      enableHiding: true,
    },
  ];
};

const WebhookActionsCell = ({ row }: { row: Row<V1Webhook> }) => {
  const { mutations, createWebhookURL } = useWebhooks(() =>
    setIsDropdownOpen(false),
  );

  const [isCopied, setIsCopied] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const webhookUrl = createWebhookURL(row.original.name);

  const handleCopy = (url: string) => {
    navigator.clipboard.writeText(url);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 1000);
    setTimeout(() => setIsDropdownOpen(false), 400);
  };

  return (
    <DropdownMenu open={isDropdownOpen} onOpenChange={setIsDropdownOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="h-8 w-8 p-0 hover:bg-muted/50">
          <DotsVerticalIcon className="h-4 w-4 text-muted-foreground cursor-pointer" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          className="flex flex-row gap-x-2"
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            handleCopy(webhookUrl);
          }}
        >
          {isCopied ? (
            <Check className="h-4 w-4 text-green-600" />
          ) : (
            <Copy className="size-4" />
          )}
          Copy Webhook URL
        </DropdownMenuItem>
        <DropdownMenuItem
          className="flex flex-row gap-x-2"
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            mutations.deleteWebhook({ webhookName: row.original.name });
          }}
          disabled={mutations.isDeletePending}
        >
          {mutations.isDeletePending ? (
            <Loader className="size-4 animate-spin" />
          ) : (
            <Trash2 className="size-4" />
          )}
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
