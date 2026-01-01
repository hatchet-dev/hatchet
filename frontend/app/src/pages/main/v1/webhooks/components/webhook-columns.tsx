import { useWebhooks } from '../hooks/use-webhooks';
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
import { Check, Copy, Loader, Save, Trash2, X } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';

export const WebhookActionsCell = ({ webhook }: { webhook: V1Webhook }) => {
  const { mutations, createWebhookURL } = useWebhooks(() =>
    setIsDropdownOpen(false),
  );
  const [isCopied, setIsCopied] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);

  const webhookUrl = createWebhookURL(webhook.name);

  const handleCopy = (url: string) => {
    navigator.clipboard.writeText(url);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 1000);
    setTimeout(() => setIsDropdownOpen(false), 400);
  };

  return (
    <DropdownMenu open={isDropdownOpen} onOpenChange={setIsDropdownOpen}>
      <DropdownMenuTrigger asChild>
        <Button variant="icon" size="sm">
          <DotsVerticalIcon className="size-4 cursor-pointer text-muted-foreground" />
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
            <Check className="size-4 text-green-600" />
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
            mutations.deleteWebhook({ webhookName: webhook.name });
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

export const EditableExpressionCell = ({ webhook }: { webhook: V1Webhook }) => {
  const { mutations } = useWebhooks();
  const [isEditing, setIsEditing] = useState(false);
  const [value, setValue] = useState(webhook.eventKeyExpression || '');

  const hasChanges =
    value.trim() !== (webhook.eventKeyExpression || '').trim() &&
    value.trim() !== '';

  // Sync value when webhook data changes (e.g., after successful save) and there are no unsaved changes
  useEffect(() => {
    if (!isEditing && !hasChanges) {
      setValue(webhook.eventKeyExpression || '');
    }
  }, [webhook.eventKeyExpression, isEditing, hasChanges]);

  const handleSave = useCallback(() => {
    if (value !== webhook.eventKeyExpression && value.trim()) {
      mutations.updateWebhook({
        webhookName: webhook.name,
        webhookData: { eventKeyExpression: value.trim() },
      });
    }
    setIsEditing(false);
  }, [value, webhook.eventKeyExpression, webhook.name, mutations]);

  const handleCancel = useCallback(() => {
    setValue(webhook.eventKeyExpression || '');
    setIsEditing(false);
  }, [webhook.eventKeyExpression, setIsEditing, setValue]);

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
          className={`h-6 w-full rounded bg-muted px-2 py-3 font-mono text-xs transition-colors ${
            isEditing || hasChanges
              ? 'cursor-text border-input focus:border-ring focus:ring-1 focus:ring-ring'
              : 'cursor-text border-transparent hover:bg-muted/80'
          }`}
          readOnly={!isEditing && !hasChanges}
          autoFocus={isEditing}
        />
      </div>
      {(isEditing || hasChanges) && (
        <div className="flex flex-row items-center duration-200 animate-in fade-in-0 slide-in-from-right-2">
          <Button
            variant="icon"
            size="icon"
            onClick={handleSave}
            className={`${
              hasChanges && !mutations.isUpdatePending
                ? 'animate-pulse text-red-500/80'
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
            variant="icon"
            size="icon"
            onClick={handleCancel}
            disabled={mutations.isUpdatePending}
          >
            <X className="size-3" />
          </Button>
        </div>
      )}
    </div>
  );
};
