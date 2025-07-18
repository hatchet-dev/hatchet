import { ColumnDef, Row } from '@tanstack/react-table';
import { V1Webhook, V1WebhookAuthType, V1WebhookSourceName } from '@/lib/api';
import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { DotsVerticalIcon, GitHubLogoIcon } from '@radix-ui/react-icons';
import {
  Check,
  Copy,
  Key,
  ShieldCheck,
  Trash2,
  UserCheck,
  Webhook,
} from 'lucide-react';
import { FaStripeS } from 'react-icons/fa';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { useState } from 'react';
import { useWebhooks } from '../hooks/use-webhooks';

const SourceName = ({ sourceName }: { sourceName: V1WebhookSourceName }) => {
  switch (sourceName) {
    case V1WebhookSourceName.GENERIC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <Webhook className="size-4" />
          Generic
        </span>
      );
    case V1WebhookSourceName.GITHUB:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <GitHubLogoIcon className="size-4" />
          GitHub
        </span>
      );
    case V1WebhookSourceName.STRIPE:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <FaStripeS className="size-4" />
          GitHub
        </span>
      );

    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = sourceName;
      throw new Error(`Unhandled source: ${exhaustiveCheck}`);
  }
};

const AuthMethod = ({ authMethod }: { authMethod: V1WebhookAuthType }) => {
  switch (authMethod) {
    case V1WebhookAuthType.BASIC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <UserCheck className="size-4" />
          Basic
        </span>
      );
    case V1WebhookAuthType.API_KEY:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <Key className="size-4" />
          API Key
        </span>
      );
    case V1WebhookAuthType.HMAC:
      return (
        <span className="flex flex-row gap-x-2 items-center">
          <ShieldCheck className="size-4" />
          HMAC
        </span>
      );

    default:
      // eslint-disable-next-line no-case-declarations
      const exhaustiveCheck: never = authMethod;
      throw new Error(`Unhandled auth method: ${exhaustiveCheck}`);
  }
};

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
  const { mutations } = useWebhooks();

  const [isCopied, setIsCopied] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const webhookUrl = `${window.location.protocol}//${window.location.hostname}/api/v1/stable/tenants/${row.original.tenantId}/webhooks/${row.original.name}`;

  const handleCopy = (url: string) => {
    navigator.clipboard.writeText(url);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 1000);
    setTimeout(() => setIsDropdownOpen(false), 400);
  };

  const handleDelete = (webhookName: string) => {
    mutations.deleteWebhook({ webhookName });
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
            handleDelete(row.original.name);
          }}
        >
          <Trash2 className="size-4" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
