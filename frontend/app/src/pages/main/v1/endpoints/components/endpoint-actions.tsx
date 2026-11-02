import { useEndpointMutations } from '../hooks/use-endpoints';
import { Copy, t } from './copy';
import { EditEndpointDialog } from './edit-endpoint-dialog';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import useCanWrite from '@/hooks/use-can-write';
import { V1ServerlessEndpoint } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { DotsVerticalIcon } from '@radix-ui/react-icons';
import { AxiosError } from 'axios';
import { Edit, Pause, Play, Trash2 } from 'lucide-react';
import { useState } from 'react';

export const EndpointActionsMenu = ({
  endpoint,
  onDeleted,
}: {
  endpoint: V1ServerlessEndpoint;
  onDeleted?: () => void;
}) => {
  const canWrite = useCanWrite();
  const { update, remove } = useEndpointMutations();
  const { handleApiError } = useApiError({});
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);

  if (!canWrite) {
    return null;
  }

  const toggleEnabled = () =>
    update.mutate(
      {
        endpointId: endpoint.metadata.id,
        data: { enabled: !endpoint.enabled },
      },
      { onError: (error) => handleApiError(error as AxiosError) },
    );

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="icon" size="sm" aria-label={t('actions.menu')}>
            <DotsVerticalIcon className="size-4 cursor-pointer text-muted-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            className="flex flex-row gap-x-2"
            onClick={() => setIsEditOpen(true)}
          >
            <Edit className="size-4" />
            {t('actions.edit')}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="flex flex-row gap-x-2"
            disabled={update.isPending}
            onClick={toggleEnabled}
          >
            {endpoint.enabled ? (
              <Pause className="size-4" />
            ) : (
              <Play className="size-4" />
            )}
            {t(endpoint.enabled ? 'actions.disable' : 'actions.enable')}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="flex flex-row gap-x-2 text-red-600"
            onClick={() => setIsDeleteOpen(true)}
          >
            <Trash2 className="size-4" />
            {t('actions.delete')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <EditEndpointDialog
        endpoint={endpoint}
        open={isEditOpen}
        onOpenChange={setIsEditOpen}
      />
      <ConfirmDialog
        isOpen={isDeleteOpen}
        title={t('delete.title', { name: endpoint.name })}
        description={<Copy k="delete.description" />}
        submitLabel={t('delete.submit')}
        isLoading={remove.isPending}
        onCancel={() => setIsDeleteOpen(false)}
        onSubmit={() =>
          remove.mutate(endpoint.metadata.id, {
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
