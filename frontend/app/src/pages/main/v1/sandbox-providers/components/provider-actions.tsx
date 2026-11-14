import { useSandboxProviderMutations } from '../hooks/use-sandbox-providers';
import { EditProviderDialog } from './edit-provider-dialog';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import useCanWrite from '@/hooks/use-can-write';
import { SandboxProvider } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { AxiosError } from 'axios';
import { Edit, Pause, Play, Trash2 } from 'lucide-react';
import { useState } from 'react';

export const ProviderActionsMenu = ({
  provider,
  onDeleted,
}: {
  provider: SandboxProvider;
  onDeleted?: () => void;
}) => {
  const canWrite = useCanWrite();
  const { update, remove } = useSandboxProviderMutations();
  const { handleApiError } = useApiError({});
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);

  if (!canWrite) {
    return null;
  }

  const toggleEnabled = () =>
    update.mutate(
      {
        providerId: provider.metadata.id,
        data: { enabled: !provider.enabled },
      },
      { onError: (error) => handleApiError(error as AxiosError) },
    );

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="icon" size="sm" aria-label="Provider actions">
            <DotsVerticalIcon className="size-4 cursor-pointer text-muted-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            className="flex flex-row gap-x-2"
            onClick={() => setIsEditOpen(true)}
          >
            <Edit className="size-4" />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem
            className="flex flex-row gap-x-2"
            disabled={update.isPending}
            onClick={toggleEnabled}
          >
            {provider.enabled ? (
              <Pause className="size-4" />
            ) : (
              <Play className="size-4" />
            )}
            {provider.enabled ? 'Disable' : 'Enable'}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="flex flex-row gap-x-2 text-red-600"
            onClick={() => setIsDeleteOpen(true)}
          >
            <Trash2 className="size-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <EditProviderDialog
        provider={provider}
        open={isEditOpen}
        onOpenChange={setIsEditOpen}
      />
      <ConfirmDialog
        isOpen={isDeleteOpen}
        title={`Delete ${provider.name}?`}
        description="Sandboxes created on this provider can no longer be reached from the SDK. The vendor account and its machines are not touched."
        submitLabel="Delete provider"
        isLoading={remove.isPending}
        onCancel={() => setIsDeleteOpen(false)}
        onSubmit={() =>
          remove.mutate(provider.metadata.id, {
            onSuccess: () => {
              setIsDeleteOpen(false);
              onDeleted?.();
            },
            onError: (error) => handleApiError(error as AxiosError),
          })
        }
      />
    </>
  );
};
