import { DataTableColumnHeader } from '@/components/v1/molecules/data-table/data-table-column-header';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { V1Webhook } from '@/lib/api';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { ColumnDef, Row } from '@tanstack/react-table';
import { Check, Copy, Loader, Save, Trash2, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { useWebhooks } from '../hooks/use-webhooks';
import { AuthMethod } from './auth-method';
import { SourceName } from './source-name';

export const WebhookColumn = {
  name: 'Name',
  sourceName: 'Source',
  expression: 'Expression',
  authType: 'Auth Method',
  actions: 'Actions',
};

export type WebhookColumnKeys = keyof typeof WebhookColumn;

export const nameKey: WebhookColumnKeys = 'name';
export const sourceNameKey: WebhookColumnKeys = 'sourceName';
export const expressionKey: WebhookColumnKeys = 'expression';
export const authTypeKey: WebhookColumnKeys = 'authType';
export const actionsKey: WebhookColumnKeys = 'actions';

export const columns = (): ColumnDef<V1Webhook>[] => {
  return [
    {
      accessorKey: nameKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={WebhookColumn.name} />
      ),
      cell: ({ row }) => <div className="w-full">{row.original.name}</div>,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: sourceNameKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={WebhookColumn.sourceName}
        />
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
      accessorKey: expressionKey,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={WebhookColumn.expression}
        />
      ),
      cell: ({ row }) => <EditableExpressionCell row={row} />,
      enableSorting: false,
      enableHiding: true,
    },
    {
      accessorKey: authTypeKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={WebhookColumn.authType} />
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
      accessorKey: actionsKey,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={WebhookColumn.actions} />
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

const EditableExpressionCell = ({ row }: { row: Row<V1Webhook> }) => {
  const { mutations } = useWebhooks();
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState(row.original.eventKeyExpression || '');

  const hasChanges =
    value.trim() !== (row.original.eventKeyExpression || '').trim() &&
    value.trim() !== '';

  // Sync value when row data changes (e.g., after successful save) and there are no unsaved changes
  useEffect(() => {
    if (!isEditing && !hasChanges) {
      setValue(row.original.eventKeyExpression || '');
    }
  }, [row.original.eventKeyExpression, isEditing, hasChanges]);

  const handleSave = useCallback(() => {
    if (value !== row.original.eventKeyExpression && value.trim()) {
      mutations.updateWebhook({
        webhookName: row.original.name,
        webhookData: { eventKeyExpression: value.trim() },
      });
      setIsEditing(false);
    } else {
      setIsEditing(false);
    }
  }, [value, row.original.eventKeyExpression, row.original.name, mutations]);

  const handleCancel = useCallback(() => {
    setValue(row.original.eventKeyExpression || '');
    setIsEditing(false);
  }, [row.original.eventKeyExpression, setIsEditing, setValue]);

  const handleBlur = useCallback(() => {
    // Only auto-save if there are no changes, otherwise keep buttons visible
    if (!hasChanges) {
      setIsEditing(false);
    }
  }, [hasChanges]);

  return (
    <div className="flex flex-row items-center gap-x-2">
      <div className="relative w-full">
        <Input
          value={value}
          onChange={(e) => {
            setValue(e.target.value);
            if (!isEditing) {
              setIsEditing(true);
            }
          }}
          onClick={!isEditing ? () => setIsEditing(true) : undefined}
          onBlur={handleBlur}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && hasChanges) {
              handleSave();
            } else if (e.key === 'Escape') {
              handleCancel();
            }
          }}
          className={`bg-muted rounded px-2 py-3 font-mono text-xs w-full h-6 transition-colors ${isEditing || hasChanges
              ? 'border-input focus:border-ring focus:ring-1 focus:ring-ring cursor-text'
              : 'border-transparent cursor-text hover:bg-muted/80'
            }`}
          readOnly={!isEditing && !hasChanges}
          autoFocus={isEditing}
        />
      </div>
      {(isEditing || hasChanges) && (
        <div className="flex flex-row items-center gap-x-2 animate-in fade-in-0 slide-in-from-right-2 duration-200">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleSave}
            className={`h-7 w-7 ${hasChanges && !mutations.isUpdatePending
                ? 'text-red-500/80 animate-pulse'
                : ''
              }`}
            disabled={!hasChanges || !value.trim() || mutations.isUpdatePending}
          >
            {mutations.isUpdatePending ? (
              <Loader className="size-3 animate-spin" />
            ) : (
              <Save className="size-3" />
            )}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={handleCancel}
            className="h-7 w-7"
            disabled={mutations.isUpdatePending}
          >
            <X className="size-3" />
          </Button>
        </div>
      )}
    </div>
  );
};
