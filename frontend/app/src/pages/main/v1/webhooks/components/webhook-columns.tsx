import { EditWebhookModal } from '../index';
import { useWebhooks } from '../hooks/use-webhooks';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { V1Webhook } from '@/lib/api';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { Check, Copy, Edit, Loader, Trash2 } from 'lucide-react';
import { useState } from 'react';

export const WebhookActionsCell = ({ webhook }: { webhook: V1Webhook }) => {
  const { mutations, createWebhookURL } = useWebhooks(() =>
    setIsDropdownOpen(false),
  );
  const [isCopied, setIsCopied] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);

  const webhookUrl = createWebhookURL(webhook.name);

  const handleCopy = (url: string) => {
    navigator.clipboard.writeText(url);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 1000);
    setTimeout(() => setIsDropdownOpen(false), 400);
  };

  return (
    <>
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
              setIsDropdownOpen(false);
              setIsEditModalOpen(true);
            }}
          >
            <Edit className="size-4" />
            Edit
          </DropdownMenuItem>
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
      <EditWebhookModal
        webhook={webhook}
        open={isEditModalOpen}
        onOpenChange={setIsEditModalOpen}
      />
    </>
  );
};
